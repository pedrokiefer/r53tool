package cli

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dig"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/spf13/cobra"
)

type checkZoneApp struct {
	Profile    string
	Domain     string
	AllDomains bool

	routeManager RouteManagerAPI
}

func init() {
	rootCmd.AddCommand(newCheckZoneCmd())
}

func (a *checkZoneApp) Run(ctx context.Context) error {
	a.routeManager = newRouteManager(ctx, a.Profile, &dns.RouteManagerOptions{
		NoWait: noWait,
	})

	if a.Domain != "" {
		zone, err := a.routeManager.GetHostedZone(ctx, a.Domain)
		if err != nil {
			return err
		}
		err = a.checkZone(ctx, zone)
		return err
	}

	if a.AllDomains {
		zones, err := a.routeManager.ListHostedZones(ctx)
		if err != nil {
			return err
		}

		for _, zone := range zones {
			err := a.checkZone(ctx, zone)
			if err != nil {
				var nsr *dig.NSRecordNotFound
				if errors.As(err, &nsr) {
					continue
				} else {
					return err
				}
			}
		}
	}

	return nil
}

func (a *checkZoneApp) checkZone(ctx context.Context, zone rtypes.HostedZone) error {
	domain := aws.ToString(zone.Name)
	zoneID := aws.ToString(zone.Id)

	log.Printf("Checking %s ...\n", domain)

	digNS, err := getNameserversFor(domain)
	if err != nil {
		log.Printf("no NS records found for %s zone %s", domain, zoneID)
		return err
	}

	r53NS, err := a.routeManager.GetNSRecords(ctx, zoneID)
	if err != nil {
		return err
	}

	if !dns.MatchNSRecords(digNS, r53NS) {
		log.Printf("%s zone %s has different NS servers:\n - nameservers: %s\n - zone record: %s",
			domain,
			zoneID,
			strings.Join(digNS, ","),
			nsRecordsToString(r53NS),
		)
		return nil
	}

	log.Printf("DNS entries and Route53 NS records match.")

	return nil
}

func newCheckZoneCmd() *cobra.Command {
	a := checkZoneApp{}

	c := &cobra.Command{
		Use:   "check-zone [-d domain | -a] profile",
		Short: "Check if a zone exists",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Profile = args[0]
			return a.Run(cmd.Context())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	f := c.Flags()
	f.BoolVar(&a.AllDomains, "a", true, "Check all domains")
	f.StringVar(&a.Domain, "d", "", "Check a specific domain")
	c.MarkFlagsMutuallyExclusive("a", "d")

	return c
}
