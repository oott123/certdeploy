package util

import (
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"github.com/oott123/certdeploy/pkg/certparser"
	"software.sslmate.com/src/go-pkcs12"
)

func CertificatesToPfx(certs []*x509.Certificate, key interface{}) ([]byte, error) {
	return pkcs12.Encode(rand.Reader, key, certs[0], certs[1:], "")
}

func PemToPfx(cert, key string) ([]byte, error) {
	certs, err := certparser.CertificatesFromPEM(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certs from pem: %w", err)
	}
	pk, err := certparser.PrivateKeyFromPem(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key from pem: %w", err)
	}
	pfx, err := CertificatesToPfx(certs, pk)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pfx: %w", err)
	}
	return pfx, nil
}
