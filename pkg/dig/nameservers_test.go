package dig

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func TestGetNameserversFor_Success(t *testing.T) {
	// Save and restore seams
	oldCfg := loadClientConfig
	oldEx := exchangeFunc
	t.Cleanup(func() { loadClientConfig = oldCfg; exchangeFunc = oldEx })

	loadClientConfig = func() (*dns.ClientConfig, error) {
		return &dns.ClientConfig{Servers: []string{"127.0.0.1"}, Port: "53"}, nil
	}
	exchangeFunc = func(c *dns.Client, m *dns.Msg, addr string) (*dns.Msg, time.Duration, error) {
		msg := new(dns.Msg)
		msg.SetReply(m)
		msg.Rcode = dns.RcodeSuccess
		// Answer with two NS records
		msg.Answer = []dns.RR{
			&dns.NS{Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 60}, Ns: "ns1.example.net."},
			&dns.NS{Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 60}, Ns: "ns2.example.net."},
		}
		return msg, 0, nil
	}

	nss, err := GetNameserversFor("example.com")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"ns1.example.net", "ns2.example.net"}, nss)
}

func TestGetNameserversFor_NotFound(t *testing.T) {
	oldCfg := loadClientConfig
	oldEx := exchangeFunc
	t.Cleanup(func() { loadClientConfig = oldCfg; exchangeFunc = oldEx })

	loadClientConfig = func() (*dns.ClientConfig, error) {
		return &dns.ClientConfig{Servers: []string{"127.0.0.1"}, Port: "53"}, nil
	}
	exchangeFunc = func(c *dns.Client, m *dns.Msg, addr string) (*dns.Msg, time.Duration, error) {
		msg := new(dns.Msg)
		msg.SetReply(m)
		msg.Rcode = dns.RcodeNameError // NXDOMAIN
		return msg, 0, nil
	}

	_, err := GetNameserversFor("example.com")
	require.Error(t, err)
	var nf *NSRecordNotFound
	require.True(t, errors.As(err, &nf))
}

// timeoutError implements net.Error and always reports a timeout
type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestGetNameserversFor_TimeoutRetriesThenFail(t *testing.T) {
	oldCfg := loadClientConfig
	oldEx := exchangeFunc
	t.Cleanup(func() { loadClientConfig = oldCfg; exchangeFunc = oldEx })

	loadClientConfig = func() (*dns.ClientConfig, error) {
		return &dns.ClientConfig{Servers: []string{"127.0.0.1"}, Port: "53"}, nil
	}

	calls := 0
	exchangeFunc = func(c *dns.Client, m *dns.Msg, addr string) (*dns.Msg, time.Duration, error) {
		calls++
		return nil, 0, timeoutError{}
	}

	_, err := GetNameserversFor("example.com")
	require.Error(t, err)
	// ensure we attempted retries (maxRetry=3 => 3 exchange attempts, then fail)
	require.Equal(t, 3, calls)

	// sanity: make sure the error is not a net.Error itself (it should be a fmt error from our function)
	var ne net.Error
	require.False(t, errors.As(err, &ne))
}
