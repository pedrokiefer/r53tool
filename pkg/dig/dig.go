package dig

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var nsre = regexp.MustCompile(`.*NS.(.*)`)

type NSRecordNotFound struct {
	Domain string
}

func (e *NSRecordNotFound) Error() string {
	return fmt.Sprintf("failed to get nameservers for: %s", e.Domain)
}

func GetNameserversFor(domain string) ([]string, error) {
	config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")

	c := &dns.Client{
		Timeout: 15 * time.Second,
	}

	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn(domain), dns.TypeNS)
	m.RecursionDesired = true

	var r *dns.Msg
	var err error
	maxRetry := 3
	retries := 0
	for {
		retries++

		if retries > maxRetry {
			return nil, fmt.Errorf("failed to get nameservers for: %s", domain)
		}

		r, _, err = c.Exchange(m, net.JoinHostPort(config.Servers[0], config.Port))
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return nil, err
		}
		break
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, &NSRecordNotFound{Domain: domain}
	}

	nss := []string{}
	for _, rr := range r.Answer {
		if ns, ok := rr.(*dns.NS); ok {
			nsStr := ns.String()
			server := nsre.FindStringSubmatch(nsStr)[1]
			server = strings.TrimSuffix(server, ".")
			nss = append(nss, server)
		}
	}
	return nss, nil
}
