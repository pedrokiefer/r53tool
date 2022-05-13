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
	"github.com/manifoldco/promptui"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/spf13/cobra"
	"inet.af/netaddr"
)

type parkApp struct {
	Profile string

	IPSv4 []rtypes.ResourceRecord
	IPSv6 []rtypes.ResourceRecord

	Hostname string
	ZoneID   string

	Domain     string
	AllDomains bool
	Alias      bool
	Force      bool

	service *dns.RouteManager
}

func init() {
	rootCmd.AddCommand(newParkCommand())
}

func (a *parkApp) Run(ctx context.Context) error {
	a.service = dns.NewRouteManager(ctx, a.Profile, &dns.RouteManagerOptions{
		NoWait: noWait,
	})
	log.Printf("Parking domains in %s...\n", a.Profile)

	if a.Domain != "" {
		zone, err := a.service.GetHostedZone(ctx, a.Domain)
		if err != nil {
			return err
		}

		err = a.parkZone(ctx, zone)
		return err
	}

	if !a.AllDomains {
		return errors.New("-a or -d is required")
	}

	zones, err := a.service.ListHostedZones(ctx)
	if err != nil {
		return err
	}

	for _, zone := range zones {
		err := a.parkZone(ctx, zone)
		if err != nil {
			log.Printf("error parking zone %s: %+v", aws.ToString(zone.Name), err)
		}
	}

	return nil
}

func (a *parkApp) parkZone(ctx context.Context, zone rtypes.HostedZone) error {
	zoneID := aws.ToString(zone.Id)
	zoneName := aws.ToString(zone.Name)
	tags, err := a.service.GetZoneTags(ctx, zoneID)
	if err != nil {
		return err
	}

	parked := hasParkedTag(tags)
	count := aws.ToInt64(zone.ResourceRecordSetCount)

	if count > 2 {
		if !a.Force {
			log.Printf("Skipping %s (has %d records)", zoneName, count)
			return nil
		}

		records, err := a.service.GetResourceRecords(ctx, zoneID)
		if err != nil {
			return err
		}
		_, pt := dns.FindParkedResourceRecord(records, zoneName)
		if (pt.HasARecord || pt.HasAAAARecord) && pt.HasWWWCnameRecord && !parked {
			prompt := promptui.Prompt{
				Label:     "[WARNING] Domain is in use. Do you want to overwrite those entries?",
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
		}
	}

	if parked {
		prompt := promptui.Prompt{
			Label:     "Domain already parked. Do you want to update those entries?",
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
	}

	log.Printf("Parking %s...\n", zoneName)
	changes := a.createChanges(aws.ToString(zone.Name))

	info, err := a.service.UpdateRecords(ctx, "parking "+zoneName, zoneID, changes)
	if err != nil {
		return err
	}

	err = a.service.WaitForChange(ctx, aws.ToString(info.Id), 1*time.Minute)
	if err != nil {
		return err
	}

	if !parked {
		err := a.service.UpsertTags(ctx, zoneID, []dns.Tag{{Name: "parked", Value: "true"}})
		if err != nil {
			return err
		}
	}
	log.Printf("Parked %s\n", zoneName)

	return nil
}

func (a *parkApp) createChanges(fqdn string) []rtypes.Change {
	changes := []rtypes.Change{
		{
			Action: rtypes.ChangeActionUpsert,
			ResourceRecordSet: &rtypes.ResourceRecordSet{
				Name: aws.String(fmt.Sprintf("www.%s", fqdn)),
				Type: rtypes.RRTypeCname,
				ResourceRecords: []rtypes.ResourceRecord{
					{Value: aws.String(fqdn)},
				},
				TTL: aws.Int64(3600),
			},
		},
	}

	if a.Alias {
		changes = append(changes, rtypes.Change{
			Action: rtypes.ChangeActionUpsert,
			ResourceRecordSet: &rtypes.ResourceRecordSet{
				Name:        aws.String(fqdn),
				Type:        rtypes.RRTypeA,
				AliasTarget: &rtypes.AliasTarget{HostedZoneId: aws.String(a.ZoneID), DNSName: aws.String(a.Hostname)},
			},
		})
		return changes
	}

	if len(a.IPSv4) > 0 {
		changes = append(changes, rtypes.Change{
			Action: rtypes.ChangeActionUpsert,
			ResourceRecordSet: &rtypes.ResourceRecordSet{
				Name:            aws.String(fqdn),
				Type:            rtypes.RRTypeA,
				ResourceRecords: a.IPSv4,
				TTL:             aws.Int64(3600),
			},
		})
	}

	if len(a.IPSv6) > 0 {
		changes = append(changes, rtypes.Change{
			Action: rtypes.ChangeActionUpsert,
			ResourceRecordSet: &rtypes.ResourceRecordSet{
				Name:            aws.String(fqdn),
				Type:            rtypes.RRTypeA,
				ResourceRecords: a.IPSv6,
				TTL:             aws.Int64(3600),
			},
		})
	}

	return changes
}

func hasParkedTag(tags []dns.Tag) bool {
	for _, tag := range tags {
		if tag.Name == "parked" && strings.ToLower(tag.Value) == "true" {
			return true
		}
	}
	return false
}

func newParkCommand() *cobra.Command {
	a := parkApp{
		IPSv4: []rtypes.ResourceRecord{},
		IPSv6: []rtypes.ResourceRecord{},
	}

	c := &cobra.Command{
		Use:   "park <profile> <destination_ips>",
		Short: "Park is a tool to park domains in Route53 creating A and www CNAME records",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Profile = args[0]

			if !a.Alias {
				for _, v := range args[1:] {
					ip, err := netaddr.ParseIP(v)
					if err != nil {
						return err
					}

					r := rtypes.ResourceRecord{
						Value: aws.String(ip.String()),
					}
					if ip.Is4() {
						a.IPSv4 = append(a.IPSv4, r)
					} else {
						a.IPSv6 = append(a.IPSv6, r)
					}
				}
			} else {
				if len(args) < 2 {
					return fmt.Errorf("alias requires a hostname and zoneID")
				}
				a.Hostname = args[1]
				a.ZoneID = args[2]
			}

			return a.Run(cmd.Context())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	f := c.Flags()
	f.BoolVar(&a.Force, "force", false, "Force park")
	f.BoolVar(&a.AllDomains, "a", true, "Check all domains")
	f.BoolVar(&a.Alias, "alias", true, "Use alias for parked domains: <hostname> <zoneId>")
	f.StringVar(&a.Domain, "d", "", "Check a specific domain")
	c.MarkFlagsMutuallyExclusive("a", "d")
	return c
}
