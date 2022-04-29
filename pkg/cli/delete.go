package cli

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	rdtypes "github.com/aws/aws-sdk-go-v2/service/route53domains/types"
	"github.com/manifoldco/promptui"
	"github.com/pedrokiefer/route53copy/pkg/dig"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/spf13/cobra"
)

type deleteApp struct {
	Profile string
	Domain  string
	Force   bool
}

func init() {
	rootCmd.AddCommand(newDeleteCommand())
}

func (a *deleteApp) Run(ctx context.Context) error {
	srcManager := dns.NewRouteManager(ctx, a.Profile)

	zone, err := srcManager.GetHostedZone(ctx, a.Domain)
	if err != nil {
		return err
	}
	srcZoneID := aws.ToString(zone.Id)

	recordSets, err := srcManager.GetResourceRecords(ctx, srcZoneID)
	if err != nil {
		return err
	}

	ns, err := dig.GetNameserversFor(a.Domain)
	if err != nil {
		var nsr *dig.NSRecordNotFound
		if errors.As(err, &nsr) {
			log.Println("No NS records found for", a.Domain)
			a.Force = true
		} else {
			return err
		}
	}
	nsRecords, err := dns.FindNSRecord(recordSets)
	if err != nil {
		return err
	}

	log.Printf("Dig returned NS servers: %s\n", strings.Join(ns, ","))
	log.Printf("Route53 has NS servers: %s\n", nsRecordsToString(nsRecords))

	if dns.MatchNSRecords(ns, nsRecords) && !a.Force {
		log.Printf("Nameservers for %s match, not deleting zone\n", a.Domain)
		return nil
	}

	recordSets = dns.RemoveResourceRecordsWithTypes(recordSets, []rtypes.RRType{rtypes.RRTypeNs, rtypes.RRTypeSoa})
	log.Printf("Found %d records for domain %s to delete\n", len(recordSets), a.Domain)
	dns.PrintResourceRecords(recordSets)

	if dryRun {
		log.Printf("Dry run...exiting\n")
		return nil
	}

	prompt := promptui.Prompt{
		Label:     "Delete all records?",
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil
	}

	if result != "y" {
		log.Printf("Aborting\n")
		return nil
	}

	if len(recordSets) > 0 {
		log.Printf("Deleting records...\n")
		drchID, err := srcManager.DeleteRecords(ctx, srcZoneID, recordSets)
		if err != nil {
			return err
		}

		err = srcManager.WaitForChange(ctx, drchID, 2*time.Minute)
		if err != nil {
			return err
		}

		log.Printf("Deleted all records for domain %s\n", a.Domain)
	} else {
		log.Printf("No records to delete for domain %s\n", a.Domain)
	}
	log.Printf("Removing zoneId %s...\n", srcZoneID)

	chID, err := srcManager.DeleteHostedZone(ctx, srcZoneID)
	if err != nil {
		return err
	}

	err = srcManager.WaitForChange(ctx, chID, 2*time.Minute)
	if err != nil {
		return err
	}

	log.Printf("Deleted zoneId %s\n", srcZoneID)

	return nil
}

func newDeleteCommand() *cobra.Command {
	a := deleteApp{}

	c := &cobra.Command{
		Use:   "delete <source_profile> <domain>",
		Short: "Delete is a tool to safely remove a zone and records from Route53",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Profile = args[0]
			a.Domain = args[1]
			return a.Run(cmd.Context())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	f := c.Flags()
	f.BoolVar(&a.Force, "force", false, "Force delete")
	return c
}

func nsToString(ns []rdtypes.Nameserver) string {
	str := []string{}
	for _, n := range ns {
		str = append(str, aws.ToString(n.Name))
	}
	return strings.Join(str, ",")
}
func nsRecordsToString(rs rtypes.ResourceRecordSet) string {
	str := []string{}
	for _, n := range rs.ResourceRecords {
		str = append(str, aws.ToString(n.Value))
	}
	return strings.Join(str, ",")
}
