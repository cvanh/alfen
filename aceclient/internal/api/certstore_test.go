package api

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type testCert struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

var serial int64

func makeCert(t *testing.T, cn string, isCA bool, parent *testCert, notBefore, notAfter time.Time) *testCert {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial++
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: cn, Organization: []string{"test"}},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		IsCA:                  isCA,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	if isCA {
		tmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	} else {
		tmpl.KeyUsage = x509.KeyUsageDigitalSignature
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	parentCert, parentKey := tmpl, key
	if parent != nil {
		parentCert, parentKey = parent.cert, parent.key
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parentCert, &key.PublicKey, parentKey)
	if err != nil {
		t.Fatal(err)
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return &testCert{cert: c, key: key}
}

func TestGetCertificateACERoot(t *testing.T) {
	root, err := GetCertificate(CertificateACECARoot)
	if err != nil {
		t.Fatal(err)
	}
	if root.Subject.CommonName != "ACE firmware WebServer CA" || !root.IsCA || !bytes.Equal(root.RawIssuer, root.RawSubject) {
		t.Fatalf("root %s CA=%v", root.Subject, root.IsCA)
	}
	if err := root.CheckSignature(root.SignatureAlgorithm, root.RawTBSCertificate, root.Signature); err != nil {
		t.Fatalf("root self-signature: %v", err)
	}
	if _, err := GetCertificate(CertificateType(7)); err == nil {
		t.Fatal("unknown type must fail")
	}
	fixture, err := os.ReadFile("../../../firmware/decompiled/ACENetwork/ICUNetwork.Certificates.webserverrootcert.pem")
	if err != nil {
		t.Skip("decompiled resource not available:", err)
	}
	if !bytes.Equal(fixture, webServerRootCertPEM) {
		t.Fatal("embedded PEM differs from the ACENetwork.dll resource")
	}
}

func TestVerifyChain(t *testing.T) {
	now := time.Now()
	ca := makeCert(t, "Test CA", true, nil, now.Add(-time.Hour), now.Add(time.Hour))
	other := makeCert(t, "Test CA", true, nil, now.Add(-time.Hour), now.Add(time.Hour)) // same name, other key
	leaf := makeCert(t, "charger", false, ca, now.Add(-time.Hour), now.Add(time.Hour))
	expired := makeCert(t, "charger", false, ca, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	selfSigned := makeCert(t, "charger", false, nil, now.Add(-time.Hour), now.Add(time.Hour))
	tampered := append([]byte(nil), leaf.cert.Raw...)
	tampered[len(tampered)-5] ^= 0xFF // inside the signature

	tests := []struct {
		name  string
		raw   [][]byte
		extra []*x509.Certificate
		ok    bool
	}{
		{"leaf + CA in extra store", [][]byte{leaf.cert.Raw}, []*x509.Certificate{ca.cert}, true},
		{"leaf + CA presented", [][]byte{leaf.cert.Raw, ca.cert.Raw}, nil, true},
		{"expired leaf (NotTimeValid accepted)", [][]byte{expired.cert.Raw}, []*x509.Certificate{ca.cert}, true},
		{"self-signed leaf (UntrustedRoot accepted)", [][]byte{selfSigned.cert.Raw}, nil, true},
		{"issuer missing (PartialChain)", [][]byte{leaf.cert.Raw}, nil, false},
		{"wrong issuer key (NotSignatureValid)", [][]byte{leaf.cert.Raw}, []*x509.Certificate{other.cert}, false},
		{"tampered signature", [][]byte{tampered}, []*x509.Certificate{ca.cert}, false},
		{"nothing presented", nil, nil, false},
		{"garbage", [][]byte{{1, 2, 3}}, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyChain(tc.raw, tc.extra)
			if (err == nil) != tc.ok {
				t.Fatalf("err = %v", err)
			}
			if err != nil && !errors.Is(err, ErrInvalidCertificate) {
				t.Fatalf("err %v does not wrap ErrInvalidCertificate", err)
			}
		})
	}
	// Against the real ACE root the test CA's leaf is a partial chain.
	if err := VerifyServerCertificate([][]byte{leaf.cert.Raw}, nil); !errors.Is(err, ErrInvalidCertificate) {
		t.Fatalf("VerifyServerCertificate = %v", err)
	}
}

func TestCertificateValidationEndToEnd(t *testing.T) {
	// httptest's certificate is self-signed: accepted like an UntrustedRoot.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()
	ref := clientFor(t, srv)
	c := NewWithCertificateValidation(ref.IP, ref.Port)
	if st, _, err := c.ExecuteWebRequest(context.Background(), "info", "", nil, ExecOptions{MaxRetries: 1}); st != ValidResponse {
		t.Fatalf("self-signed server: %v %v", st, err)
	}

	// A leaf from an unknown CA, sent without its chain, is rejected (406).
	now := time.Now()
	ca := makeCert(t, "Unknown CA", true, nil, now.Add(-time.Hour), now.Add(time.Hour))
	leaf := makeCert(t, "charger", false, ca, now.Add(-time.Hour), now.Add(time.Hour))
	bad := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	bad.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{leaf.cert.Raw}, PrivateKey: leaf.key}}}
	bad.StartTLS()
	defer bad.Close()
	ref = clientFor(t, bad)
	c = NewWithCertificateValidation(ref.IP, ref.Port)
	c.Identity = "ACE1"
	st, resp, err := c.ExecuteWebRequest(context.Background(), "info", "", nil, ExecOptions{MaxRetries: 1})
	var re *RequestError
	if st != Unsuccessful || resp.StatusCode != 406 || !errors.Is(err, ErrInvalidCertificate) || !errors.As(err, &re) ||
		re.Message != "Device 'ACE1' has an invalid or expired certificate, please contact support." {
		t.Fatalf("state=%v resp=%+v err=%v", st, resp, err)
	}
	// LoginRequest folds the same failure into HttpRequestException -> 503.
	if res := c.LoginRequest(context.Background(), 0); res != (LoginResult{StatusCode: 503, Content: "An error occurred while sending the request."}) {
		t.Fatalf("login = %+v", res)
	}
	// The default client still skips verification.
	plain := New(ref.IP, ref.Port, true)
	if st, _, err := plain.ExecuteWebRequest(context.Background(), "info", "", nil, ExecOptions{MaxRetries: 1}); st != ValidResponse {
		t.Fatalf("insecure default: %v %v", st, err)
	}
}
