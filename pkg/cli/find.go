package cli

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/spf13/cobra"
)

type findApp struct {
	Profile string
	Key     string
}

func init() {
	rootCmd.AddCommand(newFindCommand())
}

func matchInResourceRecord(r *regexp.Regexp, rr rtypes.ResourceRecordSet) bool {
	if r.MatchString(aws.ToString(rr.Name)) {
		return true
	}

	if rr.AliasTarget != nil && r.MatchString(aws.ToString(rr.AliasTarget.DNSName)) {
		return true
	}

	for _, v := range rr.ResourceRecords {
		if r.MatchString(aws.ToString(v.Value)) {
			return true
		}
	}

	return false
}

func resourceRecordsToString(rr []rtypes.ResourceRecord) string {
	vs := []string{}
	for _, v := range rr {
		vs = append(vs, aws.ToString(v.Value))
	}
	return strings.Join(vs, ", ")
}

func (a *findApp) Run(ctx context.Context) error {
	manager := dns.NewRouteManager(ctx, a.Profile, &dns.RouteManagerOptions{
		NoWait: noWait,
	})

	r, err := regexp.Compile(a.Key)
	if err != nil {
		return err
	}

	zones, err := manager.ListHostedZones(ctx)
	if err != nil {
		return err
	}

	log.Printf("Found %d zones in %s\n", len(zones), a.Profile)
	log.Printf("Fetching records...\n")

	records := map[string][]rtypes.ResourceRecordSet{}

	for _, zone := range zones {
		rs, err := manager.GetResourceRecords(ctx, aws.ToString(zone.Id))
		if err != nil {
			log.Printf("failed to list records for zone %s: %s", aws.ToString(zone.Name), err)
			continue
		}
		records[aws.ToString(zone.Name)] = rs
	}

	for zone, rs := range records {
		for _, entry := range rs {
			if matchInResourceRecord(r, entry) {
				var value string
				if entry.AliasTarget != nil {
					value = aws.ToString(entry.AliasTarget.DNSName)
				}
				if len(entry.ResourceRecords) > 0 {
					value = resourceRecordsToString(entry.ResourceRecords)
				}
				fmt.Printf("%s: %s -> %s\n", zone, aws.ToString(entry.Name), value)
			}
		}
	}

	return nil
}

func newFindCommand() *cobra.Command {
	a := findApp{}

	c := &cobra.Command{
		Use:   "find <profile> <key>",
		Short: "Search all DNS entries in an AWS account for a given key",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Profile = args[0]
			a.Key = args[1]
			return a.Run(cmd.Context())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	return c
}
