package util

import "strings"

func MatchDomain(domainInCert, domainInService string) bool {
	domainInCert = normalizeDomain(domainInCert)
	domainInService = normalizeDomain(domainInService)

	if strings.Index(domainInCert, "*") == 0 {
		i := strings.Index(domainInService, domainInCert[1:])
		if i < 0 {
			return false
		}
		sub := domainInCert[0:i]
		if strings.Contains(sub, ".") {
			// don't match wildcard
			return false
		}
	} else {
		return domainInCert == domainInService
	}
	return true
}

func normalizeDomain(domain string) string {
	return strings.ToLower(domain)
}
