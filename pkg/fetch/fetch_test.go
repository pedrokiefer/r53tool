package fetch

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

type fakeResolver struct {
	ips []string
	err error
}

func (f fakeResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return f.ips, f.err
}

func TestFetch_HTTPThenHTTPS_OK(t *testing.T) {
	httpmock.ActivateNonDefault(&cli)
	defer httpmock.DeactivateAndReset()

	// Resolve host to localhost
	r = fakeResolver{ips: []string{"127.0.0.1"}}

	// Mock HTTP and HTTPS
	httpmock.RegisterResponder("GET", "http://example.com", httpmock.NewStringResponder(200, "ok"))
	httpmock.RegisterResponder("GET", "https://example.com", httpmock.NewErrorResponder(&net.OpError{Op: "remote error"}))

	ok, err := Fetch(context.Background(), "example.com")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestFetch_HTTPError_ReturnsError(t *testing.T) {
	httpmock.ActivateNonDefault(&cli)
	defer httpmock.DeactivateAndReset()

	r = fakeResolver{ips: []string{"127.0.0.1"}}

	// HTTP returns error
	httpmock.RegisterResponder("GET", "http://bad.example.com", httpmock.NewErrorResponder(errors.New("boom")))

	ok, err := Fetch(context.Background(), "bad.example.com")
	require.Error(t, err)
	require.False(t, ok)
}

func TestCheckTCP_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	// Accept one connection asynchronously
	go func() {
		conn, _ := ln.Accept()
		time.Sleep(50 * time.Millisecond)
		if conn != nil {
			conn.Close()
		}
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	ok, err := CheckTCP(context.Background(), host, port)
	require.NoError(t, err)
	require.True(t, ok)
}
