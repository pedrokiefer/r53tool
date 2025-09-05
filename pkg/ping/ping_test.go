package ping

import (
	"context"
	"errors"
	"testing"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/stretchr/testify/require"
)

type fakeStats struct{ sent, recv int }

func (f *fakeStats) PacketsSent() int { return f.sent }
func (f *fakeStats) PacketsRecv() int { return f.recv }

type fakePinger struct {
	timeout time.Duration
	count   int
	runErr  error
	stats   *probing.Statistics
}

func (f *fakePinger) SetTimeout(d time.Duration) { f.timeout = d }
func (f *fakePinger) SetCount(n int)             { f.count = n }
func (f *fakePinger) Run() error                 { return f.runErr }
func (f *fakePinger) Statistics() *probing.Statistics {
	return f.stats
}

func TestCheck_SuccessWithPacketLoss(t *testing.T) {
	t.Cleanup(func() {
		newPinger = func(host string) (Pinger, error) { p, _ := probing.NewPinger(host); return &realPinger{Pinger: p}, nil }
	})

	stats := &probing.Statistics{}
	stats.PacketsSent = 3
	stats.PacketsRecv = 2

	newPinger = func(host string) (Pinger, error) { return &fakePinger{stats: stats}, nil }

	ok, err := Check(context.Background(), "example.com")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCheck_ErrorOnRun(t *testing.T) {
	t.Cleanup(func() {
		newPinger = func(host string) (Pinger, error) { p, _ := probing.NewPinger(host); return &realPinger{Pinger: p}, nil }
	})

	newPinger = func(host string) (Pinger, error) {
		return &fakePinger{runErr: errors.New("boom"), stats: &probing.Statistics{}}, nil
	}

	ok, err := Check(context.Background(), "example.com")
	require.Error(t, err)
	require.False(t, ok)
}
