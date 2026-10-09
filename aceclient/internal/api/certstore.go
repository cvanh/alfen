package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
)

// webServerRootCertPEM is the ICUNetwork.Certificates.webserverrootcert.pem
// manifest resource of ACENetwork.dll ("ACE firmware WebServer CA", a public
// CA certificate).
//
//go:embed webserverrootcert.pem
var webServerRootCertPEM []byte

// CertificateType ports CertificateStore.CertificateType
// (ACENetwork/ICUNetwork/CertificateStore.cs).
type CertificateType int

// CertificateACECARoot ports CertificateType.ACE_CA_ROOT.
const CertificateACECARoot CertificateType = 0

// certificateMapping ports CertificateStore.certificateMapping (CertificateMap).
var certificateMapping = map[CertificateType][]byte{
	CertificateACECARoot: webServerRootCertPEM,
}

// GetCertificate ports CertificateStore.GetCertificate.
func GetCertificate(t CertificateType) (*x509.Certificate, error) {
	data, ok := certificateMapping[t]
	if !ok {
		return nil, fmt.Errorf("api: no certificate for type %d", t)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("api: certificate resource is not PEM")
	}
	return x509.ParseCertificate(block.Bytes)
}

// VerifyServerCertificate ports ICULanDevice.ValidateServerCertificate as a
// tls.Config.VerifyPeerCertificate callback. The C# adds the ACE CA root to
// the chain's ExtraStore, builds the chain for the presented certificate and
// rejects it only for chain statuses other than NotTimeValid, NotTimeNested,
// UntrustedRoot and CtlNotTimeValid. So it accepts any chain that ends in a
// self-signed root with valid signatures (the ACE root or another one), and
// ignores expiry and the host name (sslPolicyErrors is not consulted).
// Rejections wrap ErrInvalidCertificate, which ExecuteWebRequest maps to
// 406 NotAcceptable.
//
// Deviations: revocation is not checked (Windows may report
// RevocationStatusUnknown, which the C# would reject); chains that only a
// system root completes are verified with Go's verifier, which also checks
// validity periods.
func VerifyServerCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	root, err := GetCertificate(CertificateACECARoot)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCertificate, err)
	}
	return verifyChain(rawCerts, []*x509.Certificate{root})
}

// CertificateValidationTLSConfig returns the TLS settings of the installer's
// ServicePointManager: TLS 1.2/1.3 and ValidateServerCertificate.
func CertificateValidationTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify:    true, // replaced by VerifyPeerCertificate
		VerifyPeerCertificate: VerifyServerCertificate,
		MinVersion:            tls.VersionTLS12,
	}
}

// NewWithCertificateValidation is New with the optional installer-style
// certificate validation (VerifyServerCertificate) instead of skipping
// verification. New keeps its behaviour.
func NewWithCertificateValidation(ip string, port int) *Client {
	c := New(ip, port, true)
	if tr, ok := c.HTTP.Transport.(*http.Transport); ok {
		tr.TLSClientConfig = CertificateValidationTLSConfig()
	}
	return c
}

// verifyChain walks issuer links from the leaf through the presented
// certificates and extra (the ExtraStore) up to a self-signed root.
func verifyChain(rawCerts [][]byte, extra []*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("%w: no certificate presented", ErrInvalidCertificate)
	}
	certs := make([]*x509.Certificate, len(rawCerts))
	for i, raw := range rawCerts {
		c, err := x509.ParseCertificate(raw)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidCertificate, err)
		}
		certs[i] = c
	}
	pool := append(append([]*x509.Certificate(nil), certs[1:]...), extra...)
	cur := certs[0]
	seen := map[string]bool{string(cur.Raw): true}
	for depth := 0; depth < 16; depth++ {
		if bytes.Equal(cur.RawIssuer, cur.RawSubject) {
			if err := cur.CheckSignature(cur.SignatureAlgorithm, cur.RawTBSCertificate, cur.Signature); err != nil {
				return fmt.Errorf("%w: NotSignatureValid: %v", ErrInvalidCertificate, err)
			}
			return nil // root reached: NoError or UntrustedRoot, both accepted
		}
		var next *x509.Certificate
		var sigErr error
		for _, cand := range pool {
			if !bytes.Equal(cand.RawSubject, cur.RawIssuer) {
				continue
			}
			if err := cur.CheckSignatureFrom(cand); err != nil {
				sigErr = err
				continue
			}
			next = cand
			break
		}
		if next == nil {
			if sigErr != nil {
				return fmt.Errorf("%w: NotSignatureValid: %v", ErrInvalidCertificate, sigErr)
			}
			if systemVerifies(certs) {
				return nil
			}
			return fmt.Errorf("%w: PartialChain: issuer %q not found", ErrInvalidCertificate, cur.Issuer.String())
		}
		if seen[string(next.Raw)] {
			return fmt.Errorf("%w: Cyclic", ErrInvalidCertificate)
		}
		seen[string(next.Raw)] = true
		cur = next
	}
	return fmt.Errorf("%w: chain too long", ErrInvalidCertificate)
}

// systemVerifies stands in for the Windows chain engine finding the issuer
// in the machine stores.
func systemVerifies(certs []*x509.Certificate) bool {
	inter := x509.NewCertPool()
	for _, c := range certs[1:] {
		inter.AddCert(c)
	}
	_, err := certs[0].Verify(x509.VerifyOptions{
		Intermediates: inter,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	return err == nil
}
