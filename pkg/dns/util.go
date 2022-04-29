package dns

import "strings"

func NormalizeDomain(domain string) string {
	if strings.HasSuffix(domain, ".") {
		return domain
	} else {
		return domain + "."
	}
}

func DenormalizeDomain(domain string) string {
	if strings.HasSuffix(domain, ".") {
		return domain[:len(domain)-1]
	} else {
		return domain
	}
}
