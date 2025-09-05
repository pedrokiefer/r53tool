package fetch

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

type flakyResolver struct{ calls int }

func (f *flakyResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	f.calls++
	if f.calls < 3 { // first two attempts timeout
		return nil, &net.DNSError{IsTimeout: true}
	}
	return []string{"127.0.0.1"}, nil
}

func TestLookupWithRetry_TimeoutThenSuccess(t *testing.T) {
	old := r
	t.Cleanup(func() { r = old })

	fr := &flakyResolver{}
	r = fr

	addrs, err := lookupWithRetry(context.Background(), "example.com", 5)
	require.NoError(t, err)
	require.Equal(t, []string{"127.0.0.1"}, addrs)
	require.GreaterOrEqual(t, fr.calls, 3)
}
