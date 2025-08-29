package ping

import (
	"context"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func Check(ctx context.Context, host string) (bool, error) {
	pinger, err := probing.NewPinger(host)
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
