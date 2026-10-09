// Command aceclient is a clean-room Go reimplementation of the Alfen ACE
// Service Installer's charger-facing surface: the .fwi/.tfw container formats,
// the SCN UDP discovery protocol, the HTTPS settings/control channel, and a
// candidate-key sweep for firmware payloads.
//
// It is for interoperability and firmware-format research on hardware you own.
// See README.md for scope and the honest limits of the key sweep.
package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"alfen/aceclient/internal/api"
	"alfen/aceclient/internal/fwi"
	"alfen/aceclient/internal/scn"
	"alfen/aceclient/internal/tvf"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "fwi-info":
		err = cmdFwiInfo(os.Args[2:])
	case "fwi-build":
		err = cmdFwiBuild(os.Args[2:])
	case "tvf-info":
		err = cmdTvfInfo(os.Args[2:])
	case "discover":
		err = cmdDiscover(os.Args[2:])
	case "scn-decode":
		err = cmdScnDecode(os.Args[2:])
	case "login":
		err = cmdLogin(os.Args[2:])
	case "prop-get":
		err = cmdPropGet(os.Args[2:])
	case "prop-set":
		err = cmdPropSet(os.Args[2:])
	case "firmware-upload":
		err = cmdFirmwareUpload(os.Args[2:])
	case "keytest":
		err = cmdKeytest(os.Args[2:])
	case "selftest":
		err = cmdSelftest(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `aceclient — Alfen ACE service tool, reimplemented in Go

FORMATS
  fwi-info   <file.fwi>                 parse header, verify CRC, try display-key unwrap
  fwi-build  <in.bin> <out.fwi>         wrap data as a display-resource .fwi (known key)
  tvf-info   <file.tfw>                 inspect an AHP .tfw (manifest / cert / payload)

NETWORK (charger)
  discover   [seconds]                  listen for SCN UDP broadcasts (port 36549)
  scn-decode <packet.hex>               decrypt+parse one raw SCN datagram (hex file or '-')
  login      --ip --port --user --pass [--display] [--insecure=false]
  prop-get   --ip --port ... [--ids 8311_0 | --params "cat=..&limit=500"]
  prop-set   --ip --port ... --id 8317_2 --type <sdt> --value <v> --name <n>
  firmware-upload --ip --port ... --file <fw.fwi|.tfw>

KEY RESEARCH
  keytest    --file <fwi> --keys <wordlist|-> [--derive hex|raw|md5|sha256]
             [--oracle entropy|display|magic] [--threshold 7.5] [--skip-first]
             [--magic 789c --magic-offset -1] [--include-known] [--workers N] [--all]
  selftest                              build→parse→unwrap→sweep with the known key

Flags take -flag or --flag. Run a subcommand with -h for its flags.
`)
}

// --- small flag helpers (std flag, but shared parse for connection options) ---

type connOpts struct {
	ip       string
	port     int
	user     string
	pass     string
	display  string
	insecure bool
}

func parseConn(args []string, extra func(get func(string) string)) (connOpts, map[string]string, []string, error) {
	// minimal long-opt parser: --k v or --k=v, plus positionals
	co := connOpts{port: 443, insecure: true, display: "aceclient"}
	vals := map[string]string{}
	var pos []string
	i := 0
	for i < len(args) {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			k := strings.TrimLeft(a, "-")
			v := ""
			if eq := strings.IndexByte(k, '='); eq >= 0 {
				v = k[eq+1:]
				k = k[:eq]
			} else if i+1 < len(args) && (args[i+1] == "-" || !strings.HasPrefix(args[i+1], "-")) {
				// accept a bare "-" (stdin) as a value
				v = args[i+1]
				i++
			} else {
				v = "true"
			}
			switch k {
			case "ip":
				co.ip = v
			case "port":
				co.port, _ = strconv.Atoi(v)
			case "user":
				co.user = v
			case "pass":
				co.pass = v
			case "display":
				co.display = v
			case "insecure":
				co.insecure = v != "false"
			default:
				vals[k] = v
			}
		} else {
			pos = append(pos, a)
		}
		i++
	}
	if extra != nil {
		extra(func(k string) string { return vals[k] })
	}
	return co, vals, pos, nil
}

// ------------------------------- commands -------------------------------

func cmdFwiInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: fwi-info <file.fwi>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	h, err := fwi.Parse(data)
	if err != nil {
		return err
	}
	fmt.Printf("file:        %s (%d bytes)\n", args[0], len(data))
	fmt.Printf("header:      %s\n", h)
	ok, stored, computed := fwi.VerifyCRC(data)
	fmt.Printf("header CRC:  stored=%08X computed=%08X ok=%v\n", stored, computed, ok)
	fmt.Printf("payload:     %d bytes at 0x%X\n", len(h.Payload(data)), fwi.HeaderLen)
	if h.LooksLikeDisplayResource() {
		fmt.Println("variant:     display-resource (installer-built, known key)")
		obj, err := fwi.UnwrapDisplayPayload(data)
		if err != nil {
			fmt.Printf("unwrap:      FAILED: %v\n", err)
		} else {
			fmt.Printf("unwrap:      OK, %d bytes of object stream recovered\n", len(obj))
		}
	} else {
		fmt.Println("variant:     controller firmware (device-held key) — not unwrappable host-side")
		if h.HasWrappedKey() {
			fmt.Printf("keyblock:    %s\n", hex.EncodeToString(h.KeyBlock[:16])+"...")
		}
	}
	return nil
}

func cmdFwiBuild(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: fwi-build <in.bin> <out.fwi>")
	}
	in, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	out, err := fwi.WrapDisplayPayload(in)
	if err != nil {
		return err
	}
	if err := os.WriteFile(args[1], out, 0644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes) — display-resource .fwi wrapping %d input bytes\n", args[1], len(out), len(in))
	return nil
}

func cmdTvfInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: tvf-info <file.tfw>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	r := tvf.Sniff(data)
	fmt.Printf("file:         %s (%d bytes)\n", args[0], len(data))
	fmt.Printf("manifest@%d:  %q\n", r.ManifestOffset, r.Manifest)
	fmt.Printf("cert present: %v (offset %d)\n", r.CertPresent, r.CertOffset)
	fmt.Printf("payload ~:    offset %d (first high-entropy window)\n", r.PayloadStart)
	return nil
}

func cmdDiscover(args []string) error {
	secs := 0
	if len(args) >= 1 {
		secs, _ = strconv.Atoi(args[0])
	}
	l, err := scn.Listen()
	if err != nil {
		return fmt.Errorf("listen on :%d: %w", scn.UDPPort, err)
	}
	defer l.Close()
	var until time.Time
	if secs > 0 {
		until = time.Now().Add(time.Duration(secs) * time.Second)
		fmt.Printf("listening on udp/:%d for %ds...\n", scn.UDPPort, secs)
	} else {
		fmt.Printf("listening on udp/:%d (ctrl-C to stop)...\n", scn.UDPPort)
	}
	seen := map[string]bool{}
	return l.Run(func(s *scn.Socket, from net.IP) {
		key := fmt.Sprintf("%d_%d_%s", s.UniqueID, s.SocketIndex, from)
		if seen[key] {
			return
		}
		seen[key] = true
		fmt.Printf("[%s] net=%q name=%q id=%d sock=%d/%d state=%s uid=%016X Lib=%d setpoint=%.1fA active=%.1f/%.1f/%.1fA\n",
			from, s.NetworkName, s.Name, s.Id, s.SocketIndex, s.TotalNumberOfSockets, s.State, s.UniqueID,
			s.ScnLibVersion, s.SetPointCurrent, s.ActiveCurrentL1, s.ActiveCurrentL2, s.ActiveCurrentL3)
	}, until)
}

func cmdScnDecode(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: scn-decode <packet.hex|->")
	}
	var raw []byte
	var err error
	if args[0] == "-" {
		raw, err = readAllStdin()
	} else {
		raw, err = os.ReadFile(args[0])
	}
	if err != nil {
		return err
	}
	ct, err := hex.DecodeString(strings.Join(strings.Fields(string(raw)), ""))
	if err != nil {
		return fmt.Errorf("decode hex: %w", err)
	}
	s, err := scn.ProcessPacket(ct, net.IPv4zero, time.Now())
	if err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("packet rejected (bad length/version/network name)")
	}
	fmt.Printf("%+v\n", *s)
	return nil
}

func cmdLogin(args []string) error {
	co, _, _, _ := parseConn(args, nil)
	if co.ip == "" {
		return fmt.Errorf("--ip required")
	}
	c := api.New(co.ip, co.port, co.insecure)
	if err := c.Login(api.LoginData{Username: co.user, Password: co.pass, DisplayName: co.display}); err != nil {
		return err
	}
	fmt.Printf("login OK; access token len=%d refresh len=%d\n", len(c.AccessToken), len(c.RefreshToken))
	return nil
}

func cmdPropGet(args []string) error {
	co, vals, _, _ := parseConn(args, nil)
	if co.ip == "" {
		return fmt.Errorf("--ip required")
	}
	c := api.New(co.ip, co.port, co.insecure)
	if co.user != "" {
		if err := c.Login(api.LoginData{Username: co.user, Password: co.pass, DisplayName: co.display}); err != nil {
			return err
		}
	}
	params := vals["params"]
	if ids := vals["ids"]; ids != "" {
		params = "ids=" + ids
	}
	if params == "" {
		params = "limit=500"
	}
	props, err := c.ReadProperties(params)
	if err != nil {
		return err
	}
	for _, p := range props {
		fmt.Printf("%-9s type=%-2d ro=%-5v cat=%-8s %q = %s\n", p.IDSub(), p.DataType, p.ReadOnly, p.Category, p.Name, p.Value)
	}
	fmt.Printf("(%d properties)\n", len(props))
	return nil
}

func cmdPropSet(args []string) error {
	co, vals, _, _ := parseConn(args, nil)
	if co.ip == "" || vals["id"] == "" {
		return fmt.Errorf("--ip and --id (IDX_SUB) required")
	}
	parts := strings.Split(vals["id"], "_")
	if len(parts) != 2 {
		return fmt.Errorf("--id must be IDX_SUB hex, e.g. 8317_2")
	}
	idx, _ := strconv.ParseUint(parts[0], 16, 16)
	sub, _ := strconv.ParseUint(parts[1], 16, 8)
	t, _ := strconv.Atoi(vals["type"])
	c := api.New(co.ip, co.port, co.insecure)
	if co.user != "" {
		if err := c.Login(api.LoginData{Username: co.user, Password: co.pass, DisplayName: co.display}); err != nil {
			return err
		}
	}
	p := api.Property{
		ID: uint16(idx), Sub: byte(sub), Name: vals["name"],
		DataType: api.SDT(t), Value: vals["value"],
	}
	if p.Name == "" {
		p.Name = p.IDSub()
	}
	if err := c.StoreProperties(p); err != nil {
		return err
	}
	fmt.Printf("stored %s = %s\n", p.IDSub(), p.Value)
	return nil
}

func cmdFirmwareUpload(args []string) error {
	co, vals, _, _ := parseConn(args, nil)
	if co.ip == "" || vals["file"] == "" {
		return fmt.Errorf("--ip and --file required")
	}
	data, err := os.ReadFile(vals["file"])
	if err != nil {
		return err
	}
	c := api.New(co.ip, co.port, co.insecure)
	if co.user != "" {
		if err := c.Login(api.LoginData{Username: co.user, Password: co.pass, DisplayName: co.display}); err != nil {
			return err
		}
	}
	fmt.Printf("uploading %s (%d bytes) verbatim to %s...\n", vals["file"], len(data), co.ip)
	resp, err := c.UploadFirmware(data)
	if err != nil {
		return err
	}
	fmt.Printf("HTTP %d: %s\n", resp.StatusCode, resp.Body)
	return nil
}

func readAllStdin() ([]byte, error) {
	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			if err.Error() == "EOF" {
				return []byte(b.String()), nil
			}
			return []byte(b.String()), nil
		}
		if n == 0 {
			return []byte(b.String()), nil
		}
	}
}
