package dig

import (
	"context"
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

func (e NSRecordNotFound) Error() string {
	return fmt.Sprintf("failed to get nameservers for: %s", e.Domain)
}

type ResolveError struct {
	Domain string
	Type   string
}

func (e ResolveError) Error() string {
	return fmt.Sprintf("failed to get nameservers for: %s", e.Domain)
}

// DNSResolver defines an interface for DNS queries used by this package.
// It enables overriding in tests.
type DNSResolver interface {
	Resolve(ctx context.Context, domain string, t string) error
}

// CurrentResolver is the pluggable resolver used by Resolve.
// It can be overridden in tests.
var CurrentResolver DNSResolver = realResolver{}

// Resolve is the public entry point which delegates to CurrentResolver.
func Resolve(ctx context.Context, domain string, t string) error {
	return CurrentResolver.Resolve(ctx, domain, t)
}

// RealResolverForTest returns a new instance of the production resolver for test restoration.
func RealResolverForTest() DNSResolver { return realResolver{} }

// test seams for GetNameserversFor
var loadClientConfig = func() (*dns.ClientConfig, error) { return dns.ClientConfigFromFile("/etc/resolv.conf") }
var exchangeFunc = func(c *dns.Client, m *dns.Msg, addr string) (*dns.Msg, time.Duration, error) {
	return c.Exchange(m, addr)
}

// realResolver contains the production implementation of DNS resolution.
type realResolver struct{}

func (realResolver) Resolve(ctx context.Context, domain string, t string) error {
	_t, ok := dns.StringToType[t]
	if !ok {
		return fmt.Errorf("invalid type: %s", t)
	}

	config := &dns.ClientConfig{
		Servers: []string{"8.8.8.8", "8.8.4.4"},
		Search:  []string{""},
		Port:    "53",
		Timeout: 5,
	}
	// config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")

	c := &dns.Client{
		Timeout: 15 * time.Second,
	}

	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn(domain), _t)
	m.RecursionDesired = true

	var r *dns.Msg
	var err error
	maxRetry := 3
	retries := 0
	for {
		retries++

		if retries > maxRetry {
			return &ResolveError{Domain: domain, Type: "timeout"}
		}

		r, _, err = c.ExchangeContext(ctx, m, net.JoinHostPort(config.Servers[0], config.Port))
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return err
		}
		break
	}

	// log.Printf("d: %s, t: %s, r.Rcode: %d", domain, t, r.Rcode)

	if r.Rcode != dns.RcodeSuccess {
		code := dns.RcodeToString[r.Rcode]
		return &ResolveError{Domain: domain, Type: code}
	}

	return nil
}

func GetNameserversFor(domain string) ([]string, error) {
	config, _ := loadClientConfig()

	c := &dns.Client{Timeout: 15 * time.Second}

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

		r, _, err = exchangeFunc(c, m, net.JoinHostPort(config.Servers[0], config.Port))
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
