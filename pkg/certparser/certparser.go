package certparser

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func CertificatesFromPEM(certPem string) (certs []*x509.Certificate, err error) {
	input := []byte(certPem)
	for {
		block, remain := pem.Decode(input)
		if block == nil {
			return
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("x509 parse failed: %w", err)
		}
		certs = append(certs, cert)
		input = remain
	}
}

func PrivateKeyFromPem(keyPem string) (interface{}, error) {
	block, _ := pem.Decode([]byte(keyPem))
	if block == nil {
		return nil, fmt.Errorf("failed to decode private key")
	}
	if block.Type == "RSA PRIVATE KEY" {
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key as rsa: %w", err)
		}
		return key, nil
	}
	if block.Type == "EC PRIVATE KEY" {
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key as ec: %w", err)
		}
		return key, nil
	}
	if block.Type == "PRIVATE KEY" {
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key as pkcs8: %w", err)
		}
		return key, nil
	}
	return nil, fmt.Errorf("unkown block type: %s", block.Type)
}

func DomainsFromCert(certPem string) ([]string, error) {
	block, _ := pem.Decode([]byte(certPem))
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("x509 parse failed: %w", err)
	}

	domains := make(map[string]bool)
	domains[cert.Subject.CommonName] = true
	for _, domain := range cert.DNSNames {
		domains[domain] = true
	}

	uniqueDomains := make([]string, 0, len(domains))
	for domain := range domains {
		uniqueDomains = append(uniqueDomains, domain)
	}

	return uniqueDomains, nil
}
