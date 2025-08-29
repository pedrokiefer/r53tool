package ping

import (
	"context"
	"time"

	//nolint:staticcheck // go-ping/ping is deprecated upstream; kept for current functionality until replacement is chosen
	"github.com/go-ping/ping"
)

func Check(ctx context.Context, host string) (bool, error) {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return false, err
	}
	pinger.Timeout = 15 * time.Second
	pinger.Count = 3
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return false, err
	}
	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats
	return stats.PacketsSent-stats.PacketsRecv > 0, nil
}
