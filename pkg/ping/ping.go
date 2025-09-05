package ping

import (
	"context"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Pinger is an abstraction over probing.Pinger to enable testing.
type Pinger interface {
	SetTimeout(d time.Duration)
	SetCount(n int)
	Run() error
	Statistics() *probing.Statistics
}

type realPinger struct{ *probing.Pinger }

func (rp *realPinger) SetTimeout(d time.Duration) { rp.Timeout = d }
func (rp *realPinger) SetCount(n int)             { rp.Count = n }

// newPinger constructs a Pinger. Overridable in tests.
var newPinger = func(host string) (Pinger, error) {
	p, err := probing.NewPinger(host)
	if err != nil {
		return nil, err
	}
	return &realPinger{Pinger: p}, nil
}

func Check(ctx context.Context, host string) (bool, error) {
	pinger, err := newPinger(host)
	if err != nil {
		return false, err
	}
	// pro-bing doesn't use context directly here; timeout covers it.
	pinger.SetTimeout(15 * time.Second)
	pinger.SetCount(3)
	if err := pinger.Run(); err != nil {
		return false, err
	}
	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats
	return stats.PacketsSent-stats.PacketsRecv > 0, nil
}
