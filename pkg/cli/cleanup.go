package cli

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/pedrokiefer/route53copy/pkg/fetch"
	"github.com/pedrokiefer/route53copy/pkg/ping"
	"github.com/spf13/cobra"
)

type cleanupZoneApp struct {
	Profile string
	Domain  string

	routeManager *dns.RouteManager
}

func init() {
	rootCmd.AddCommand(newCleanupZoneCmd())
}

func (a *cleanupZoneApp) Run(ctx context.Context) error {

	a.routeManager = dns.NewRouteManager(ctx, a.Profile, &dns.RouteManagerOptions{
		NoWait: noWait,
	})

	zone, err := a.routeManager.GetHostedZone(ctx, a.Domain)
	if err != nil {
		return err
	}
	records, err := a.routeManager.GetResourceRecords(ctx, aws.ToString(zone.Id))
	if err != nil {
		return err
	}

	toDelete := []types.ResourceRecordSet{}
	var wg sync.WaitGroup
	for _, r := range records {
		wg.Add(1)
		go func(r types.ResourceRecordSet) {
			defer wg.Done()
			valid, err := a.checkRecord(ctx, r)
			if err != nil {
				log.Printf("Error verifing record %s: %+v", aws.ToString(r.Name), err)
			}
			if !valid {
				log.Printf("Record %s can be deleted", aws.ToString(r.Name))
				toDelete = append(toDelete, r)
			}
		}(r)
	}
	wg.Wait()

	dns.PrintResourceRecords(toDelete)

	return nil
}

func (a *cleanupZoneApp) checkRecord(ctx context.Context, r types.ResourceRecordSet) (bool, error) {
	switch r.Type {
	case types.RRTypeSoa:
		fallthrough
	case types.RRTypeNs:
		fallthrough
	case types.RRTypeTxt:
		return true, nil
	case types.RRTypeCname:
		fallthrough
	case types.RRTypeA:
		fallthrough
	case types.RRTypeAaaa:
		return a.checkARecord(ctx, r)
	}

	log.Printf("Unsupported record type %s %s", string(r.Type), aws.ToString(r.Name))
	return true, nil
}

func (a *cleanupZoneApp) checkARecord(ctx context.Context, r types.ResourceRecordSet) (bool, error) {
	if strings.Contains(aws.ToString(r.Name), "._domainkey.") {
		// Skip domain keys
		return true, nil
	}

	if r.AliasTarget != nil {
		return a.checkHost(ctx, aws.ToString(r.AliasTarget.DNSName))
	}
	for _, v := range r.ResourceRecords {
		target := aws.ToString(v.Value)

		if strings.Contains(target, "acm-validations.aws") {
			// ACM Validation is a CNAME to a AWS managed TXT record
			continue
		}

		valid, err := a.checkHost(ctx, target)
		if err != nil {
			return false, err
		}
		if !valid {
			return false, nil
		}
	}
	return true, nil
}

type check struct {
	CheckFunction func(ctx context.Context, host string) (bool, error)
	Valid         bool
	Error         error
}

func checkRDS(ctx context.Context, host string) (bool, error) {
	mv, err := fetch.CheckTCP(ctx, host, 3306)
	if err != nil {
		if !strings.Contains(err.Error(), "timeout") {
			return false, err
		}
	}

	pv, err := fetch.CheckTCP(ctx, host, 5432)
	if err != nil {
		if !strings.Contains(err.Error(), "timeout") {
			return false, err
		}
	}

	// log.Printf("%s RDS Check: %t %t", host, mv, pv)

	return pv || mv, nil
}

func checkCache(ctx context.Context, host string) (bool, error) {
	rv, err := fetch.CheckTCP(ctx, host, 6379)
	if err != nil {
		if !strings.Contains(err.Error(), "timeout") {
			return false, err
		}
	}
	// log.Printf("%s ElastiCache Check: %t", host, rv)
	return rv, nil
}

func (a *cleanupZoneApp) checkHost(ctx context.Context, host string) (bool, error) {
	checkers := []*check{
		{CheckFunction: ping.Check},
	}

	switch {
	case strings.Contains(host, "rds"):
		checkers = append(checkers, &check{CheckFunction: checkRDS})
	case strings.Contains(host, "cache.amazonaws"):
		checkers = append(checkers, &check{CheckFunction: checkCache})
	default:
		checkers = append(checkers, &check{CheckFunction: fetch.Fetch})
	}

	var wg sync.WaitGroup
	for _, c := range checkers {
		wg.Add(1)
		go func(c *check) {
			defer wg.Done()
			c.Valid, c.Error = c.CheckFunction(ctx, host)
		}(c)
	}

	wg.Wait()

	for _, c := range checkers {
		if c.Valid {
			return true, nil
		}
	}

	return false, nil
}

func newCleanupZoneCmd() *cobra.Command {
	a := cleanupZoneApp{}

	c := &cobra.Command{
		Use:   "cleanup-zone <profile> <domain>",
		Short: "Check if a zone exists",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Profile = args[0]
			a.Domain = args[1]
			return a.Run(cmd.Context())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	return c
}
