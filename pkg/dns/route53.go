package dns

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/route53domains"
	rdtypes "github.com/aws/aws-sdk-go-v2/service/route53domains/types"
)

type RouteManager struct {
	cli     *route53.Client
	domains *route53domains.Client

	o *RouteManagerOptions
}

type RouteManagerOptions struct {
	NoWait bool
}

type HostedZoneNotFound struct {
	Zone string
}

func (e *HostedZoneNotFound) Error() string {
	return fmt.Sprintf("hosted zone not found: %s", e.Zone)
}

func NewRouteManager(ctx context.Context, profile string, rmo *RouteManagerOptions) *RouteManager {
	if r := os.Getenv("AWS_REGION"); r == "" {
		_ = os.Setenv("AWS_REGION", "us-east-1")
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
		config.WithRetryer(func() aws.Retryer {
			return retry.NewAdaptiveMode(func(amo *retry.AdaptiveModeOptions) {
				amo.StandardOptions = []func(*retry.StandardOptions){
					func(so *retry.StandardOptions) {
						so.MaxAttempts = 5
						so.MaxBackoff = 60 * time.Second
						so.Backoff = retry.NewExponentialJitterBackoff(so.MaxBackoff)
					},
				}
			})
		}),
	)
	if err != nil {
		panic(err)
	}

	o := &RouteManagerOptions{NoWait: false}
	if rmo != nil {
		o = rmo
	}

	return &RouteManager{
		cli:     route53.NewFromConfig(cfg),
		domains: route53domains.NewFromConfig(cfg),

		o: o,
	}
}

func (r *RouteManager) GetHostedZone(ctx context.Context, domain string) (rtypes.HostedZone, error) {
	params := &route53.ListHostedZonesByNameInput{
		DNSName:  aws.String(domain),
		MaxItems: aws.Int32(1),
	}
	resp, err := r.cli.ListHostedZonesByName(ctx, params)
	if err != nil {
		return rtypes.HostedZone{}, err
	}

	if len(resp.HostedZones) == 0 {
		return rtypes.HostedZone{}, &HostedZoneNotFound{Zone: domain}
	}

	zone := resp.HostedZones[0]
	if *zone.Name != NormalizeDomain(domain) {
		return rtypes.HostedZone{}, &HostedZoneNotFound{Zone: domain}
	}
	return zone, nil
}

func (r *RouteManager) ListHostedZones(ctx context.Context) ([]rtypes.HostedZone, error) {
	paginator := route53.NewListHostedZonesPaginator(r.cli, &route53.ListHostedZonesInput{})

	zones := []rtypes.HostedZone{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return zones, err
		}
		zones = append(zones, page.HostedZones...)
	}

	return zones, nil
}

func (r *RouteManager) CreateZone(ctx context.Context, domain string) (rtypes.HostedZone, error) {
	params := &route53.CreateHostedZoneInput{
		Name:            aws.String(NormalizeDomain(domain)),
		CallerReference: aws.String(fmt.Sprintf("%s-%d", domain, time.Now().Unix())),
		HostedZoneConfig: &rtypes.HostedZoneConfig{
			Comment:     aws.String("Created by route53copy"),
			PrivateZone: false,
		},
	}
	resp, err := r.cli.CreateHostedZone(ctx, params)
	if err != nil {
		return rtypes.HostedZone{}, err
	}

	if resp.ChangeInfo.Status != rtypes.ChangeStatusInsync {
		start := time.Now()
		err := r.WaitForChange(ctx, aws.ToString(resp.ChangeInfo.Id), 1*time.Minute)
		if err != nil {
			return *resp.HostedZone, fmt.Errorf("error waiting for change to be in-sync: %s", err)
		}
		log.Printf("Waited %s for zone '%s' to be in-sync", time.Since(start), domain)

		zone, err := r.cli.GetHostedZone(ctx, &route53.GetHostedZoneInput{
			Id: resp.HostedZone.Id,
		})
		if err != nil {
			return *resp.HostedZone, fmt.Errorf("error getting zone after change: %s", err)
		}
		return *zone.HostedZone, nil
	}

	return *resp.HostedZone, nil
}

func (r *RouteManager) WaitForChange(ctx context.Context, changeId string, maxWait time.Duration) error {
	if r.o.NoWait {
		return nil
	}
	waiter := route53.NewResourceRecordSetsChangedWaiter(r.cli, func(rrscwo *route53.ResourceRecordSetsChangedWaiterOptions) {
		rrscwo.MinDelay = 15 * time.Second
	})
	return waiter.Wait(ctx, &route53.GetChangeInput{
		Id: aws.String(changeId),
	}, maxWait)
}

func (r *RouteManager) GetOrCreateZone(ctx context.Context, domain string) (rtypes.HostedZone, error) {
	var zone rtypes.HostedZone
	var err error
	zone, err = r.GetHostedZone(ctx, domain)
	if err != nil {
		var e *HostedZoneNotFound
		if errors.As(err, &e) {
			log.Printf("Destination profile does not contain %s, creating it\n", domain)
			zone, err = r.CreateZone(ctx, domain)
			if err != nil {
				return zone, err
			}
		} else {
			return zone, err
		}
	}
	return zone, nil
}

type Tag struct {
	Name  string
	Value string
}

func (r *RouteManager) GetZoneTags(ctx context.Context, zoneID string) ([]Tag, error) {

	resourceId := zoneID
	if strings.Contains(zoneID, "/") {
		s := strings.Split(zoneID, "/")
		resourceId = s[len(s)-1]
	}

	t, err := r.cli.ListTagsForResource(ctx, &route53.ListTagsForResourceInput{
		ResourceId:   aws.String(resourceId),
		ResourceType: rtypes.TagResourceTypeHostedzone,
	})
	if err != nil {
		return nil, err
	}

	tags := []Tag{}
	for _, tag := range t.ResourceTagSet.Tags {
		tags = append(tags, Tag{
			Name:  aws.ToString(tag.Key),
			Value: aws.ToString(tag.Value),
		})
	}
	return tags, nil
}

func (r *RouteManager) UpsertTags(ctx context.Context, zoneID string, tags []Tag) error {
	resourceId := zoneID
	if strings.Contains(zoneID, "/") {
		s := strings.Split(zoneID, "/")
		resourceId = s[len(s)-1]
	}

	_, err := r.cli.ChangeTagsForResource(ctx, &route53.ChangeTagsForResourceInput{
		ResourceId:   aws.String(resourceId),
		ResourceType: rtypes.TagResourceTypeHostedzone,
		AddTags:      toAwsTags(tags),
	})
	return err
}

func toAwsTags(tags []Tag) []rtypes.Tag {
	awsTags := []rtypes.Tag{}
	for _, tag := range tags {
		awsTags = append(awsTags, rtypes.Tag{
			Key:   aws.String(tag.Name),
			Value: aws.String(tag.Value),
		})
	}
	return awsTags
}

func (r *RouteManager) GetResourceRecords(ctx context.Context, zoneId string) ([]rtypes.ResourceRecordSet, error) {
	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneId),
	}
	paginator := NewListResourceRecordSetsPaginator(r.cli, params)

	records := []rtypes.ResourceRecordSet{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return records, err
		}
		records = append(records, page.ResourceRecordSets...)
	}

	return records, nil
}

func (r *RouteManager) DeleteRecords(ctx context.Context, zoneId string, records []rtypes.ResourceRecordSet) (string, error) {
	changes := []rtypes.Change{}
	for _, record := range records {
		if record.Type == rtypes.RRTypeNs || record.Type == rtypes.RRTypeSoa {
			continue
		}
		changes = append(changes, rtypes.Change{
			Action: rtypes.ChangeActionDelete,
			ResourceRecordSet: &rtypes.ResourceRecordSet{
				Name:                    record.Name,
				Type:                    record.Type,
				AliasTarget:             record.AliasTarget,
				Failover:                record.Failover,
				GeoLocation:             record.GeoLocation,
				HealthCheckId:           record.HealthCheckId,
				MultiValueAnswer:        record.MultiValueAnswer,
				Region:                  record.Region,
				ResourceRecords:         record.ResourceRecords,
				SetIdentifier:           record.SetIdentifier,
				TTL:                     record.TTL,
				TrafficPolicyInstanceId: record.TrafficPolicyInstanceId,
				Weight:                  record.Weight,
			},
		})
	}
	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneId),
		ChangeBatch: &rtypes.ChangeBatch{
			Changes: changes,
		},
	}
	ch, err := r.cli.ChangeResourceRecordSets(ctx, params)
	if err != nil {
		return "", err
	}
	return aws.ToString(ch.ChangeInfo.Id), nil
}

func (r *RouteManager) DeleteHostedZone(ctx context.Context, zoneId string) (string, error) {
	dhz, err := r.cli.DeleteHostedZone(ctx, &route53.DeleteHostedZoneInput{
		Id: aws.String(zoneId),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(dhz.ChangeInfo.Id), nil
}

func (r *RouteManager) GetNSRecords(ctx context.Context, zoneId string) (rtypes.ResourceRecordSet, error) {
	records, err := r.GetResourceRecords(ctx, zoneId)
	if err != nil {
		return rtypes.ResourceRecordSet{}, err
	}

	for _, r := range records {
		if r.Type != rtypes.RRTypeNs {
			continue
		}
		return r, nil
	}

	return rtypes.ResourceRecordSet{}, fmt.Errorf("no NS records found")
}

func (r *RouteManager) CreateChanges(domain string, recordSets []rtypes.ResourceRecordSet) []rtypes.Change {
	domain = NormalizeDomain(domain)
	var changes []rtypes.Change
	for _, recordSet := range recordSets {
		if (recordSet.Type == rtypes.RRTypeNs || recordSet.Type == rtypes.RRTypeSoa) && *recordSet.Name == domain {
			continue
		}
		change := rtypes.Change{
			Action: rtypes.ChangeActionUpsert,
			ResourceRecordSet: &rtypes.ResourceRecordSet{
				Name:                    recordSet.Name,
				Type:                    recordSet.Type,
				AliasTarget:             recordSet.AliasTarget,
				Failover:                recordSet.Failover,
				GeoLocation:             recordSet.GeoLocation,
				HealthCheckId:           recordSet.HealthCheckId,
				MultiValueAnswer:        recordSet.MultiValueAnswer,
				Region:                  recordSet.Region,
				ResourceRecords:         recordSet.ResourceRecords,
				SetIdentifier:           recordSet.SetIdentifier,
				TTL:                     recordSet.TTL,
				TrafficPolicyInstanceId: recordSet.TrafficPolicyInstanceId,
				Weight:                  recordSet.Weight,
			},
		}
		changes = append(changes, change)
	}
	return changes

}

func (r *RouteManager) UpdateRecords(ctx context.Context, comment, zoneId string, changes []rtypes.Change) (*rtypes.ChangeInfo, error) {
	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneId),
		ChangeBatch: &rtypes.ChangeBatch{
			Changes: changes,
			Comment: aws.String(comment),
		},
	}
	resp, err := r.cli.ChangeResourceRecordSets(ctx, params)
	if err != nil {
		return nil, err
	}
	return resp.ChangeInfo, nil
}

func (r *RouteManager) UpdateNSRecords(ctx context.Context, domain, zoneId string) (bool, error) {
	nsRecords, err := r.GetNSRecords(ctx, zoneId)
	if err != nil {
		return false, err
	}
	ddo, err := r.domains.GetDomainDetail(ctx, &route53domains.GetDomainDetailInput{
		DomainName: aws.String(domain),
	})
	if err != nil {
		return false, err
	}

	nss := []string{}
	for _, n := range ddo.Nameservers {
		nss = append(nss, aws.ToString(n.Name))
	}

	if MatchNSRecords(nss, nsRecords) {
		return false, nil
	}

	newNS := nameserversFromRecords(nsRecords)

	udno, err := r.domains.UpdateDomainNameservers(ctx, &route53domains.UpdateDomainNameserversInput{
		DomainName:  aws.String(domain),
		Nameservers: newNS,
	})

	if err != nil {
		return false, err
	}
	log.Printf("Updated NS records for %s: %s", domain, aws.ToString(udno.OperationId))
	return true, nil
}

func MatchNSRecords(ns []string, rs rtypes.ResourceRecordSet) bool {
	for _, r := range rs.ResourceRecords {
		recordName := DenormalizeDomain(aws.ToString(r.Value))
		if !findInList(ns, recordName) {
			return false
		}
	}
	return true
}

func findInList(ns []string, name string) bool {
	for _, n := range ns {
		if n == name {
			return true
		}
	}
	return false
}

func nameserversFromRecords(rs rtypes.ResourceRecordSet) []rdtypes.Nameserver {
	var ns []rdtypes.Nameserver
	for _, r := range rs.ResourceRecords {
		ns = append(ns, rdtypes.Nameserver{
			Name: aws.String(DenormalizeDomain(aws.ToString(r.Value))),
		})
	}
	return ns
}
