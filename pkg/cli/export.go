package cli

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/spf13/cobra"
)

type exportApp struct {
	Profile string
	Zone    string
	Output  string
}

func init() {
	rootCmd.AddCommand(newExportCommand())
}

func (a *exportApp) Run(ctx context.Context) error {
	manager := newRouteManager(ctx, a.Profile, &dns.RouteManagerOptions{NoWait: noWait})

	zone, err := manager.GetHostedZone(ctx, a.Zone)
	if err != nil {
		return err
	}

	records, err := manager.GetResourceRecords(ctx, aws.ToString(zone.Id))
	if err != nil {
		return err
	}

	if a.Output == "" {
		name := dns.DenormalizeDomain(aws.ToString(zone.Name))
		a.Output = fmt.Sprintf("%s-%s.zone", name, time.Now().Format("20060102-150405"))
	}

	log.Printf("Exporting zone %s to %s (records: %d)\n", aws.ToString(zone.Name), a.Output, len(records))

	if dryRun {
		log.Printf("--dry provided; not writing file.\n")
		return nil
	}

	if err := writeBindZoneFile(a.Output, aws.ToString(zone.Name), records); err != nil {
		return err
	}
	log.Printf("Zone file written to %s\n", a.Output)
	return nil
}

func newExportCommand() *cobra.Command {
	a := &exportApp{}
	c := &cobra.Command{
		Use:   "export <profile> <zone>",
		Short: "Export a Route53 zone to a BIND 9 zone file",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Profile = args[0]
			a.Zone = args[1]
			return a.Run(cmd.Context())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	f := c.Flags()
	f.StringVarP(&a.Output, "output", "o", "", "Output file path for the BIND zone (default: <zone>-<timestamp>.zone)")
	return c
}
