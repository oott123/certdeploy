package certparser

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

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