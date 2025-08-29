package fetch

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/dnscache"
)

var r = &dnscache.Resolver{}

func lookupWithRetry(ctx context.Context, host string, retries int) (addrs []string, err error) {
	for i := 0; i < retries; i++ {
		addrs, err = r.LookupHost(ctx, host)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
		}
		return
	}
	return nil, err

}

func httpDialContext(ctx context.Context, network string, addr string) (conn net.Conn, err error) {
	// addr has form host:port and the port is always present
	colonPos := strings.LastIndexByte(addr, ':')
	host := addr[:colonPos]

	ips, err := lookupWithRetry(ctx, host, 5)
	if err != nil {
		return nil, err
	}

	port := addr[colonPos+1:]
	for _, ip := range ips {
		var dialer net.Dialer
		conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
		if err == nil {
			break
		}
	}
	return
}

var cli = http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DialContext: httpDialContext,
	},
}

func Fetch(ctx context.Context, host string) (bool, error) {
	url := "http://" + host
	dr, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	urls := "https://" + host
	sr, err := http.NewRequestWithContext(ctx, "GET", urls, nil)
	if err != nil {
		return false, err
	}

	_, err = cli.Do(dr)
	if err != nil {
		// log.Printf("Error fetching %s: %+v", url, err)
		return false, err
	}

	_, err = cli.Do(sr)
	if err != nil {
		var oe *net.OpError
		if errors.As(err, &oe) {
			if oe.Op != "remote error" {
				return false, err
			}
		} else {
			// log.Printf("Error fetching %s: %+v", url, err)
			return false, err
		}
	}

	return true, nil
}

func CheckTCP(ctx context.Context, host string, port int) (bool, error) {
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.Close() }()
	return true, nil
}
