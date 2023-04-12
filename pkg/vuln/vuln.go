package vuln

import (
	"context"
	"log"

	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func Scan(ctx context.Context, zm ZoneMeta, rs []rtypes.ResourceRecordSet) *Findings {
	f := NewFindings(zm)
	log.Printf("Checking zone %s:\n", WhiteBold.Sprintf(zm.Name))
	log.Printf(" - %s...\n", WhiteBold.Sprintf("Checking mail vulnerabilities"))
	MailCheck(ctx, f, rs)
	log.Printf(" - %s...\n", WhiteBold.Sprintf("Checking subdomain takeover"))
	for _, entry := range rs {
		SubDomainTakeoverCheck(ctx, f, entry)
	}
	return f
}
