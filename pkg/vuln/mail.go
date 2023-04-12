package vuln

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func MailCheck(ctx context.Context, f *Findings, rs []rtypes.ResourceRecordSet) {
	mx := findByType(rs, rtypes.RRTypeMx)
	if len(mx) == 0 {
		return
	}

	for _, m := range mx {
		name := aws.ToString(m.Name)

		CheckSPF(name, rs)
		CheckDMARC(name, rs)
	}

}

func findByTypeAndName(rs []rtypes.ResourceRecordSet, t rtypes.RRType, name string) []rtypes.ResourceRecordSet {
	var result []rtypes.ResourceRecordSet
	for _, r := range rs {
		if r.Type == t && aws.ToString(r.Name) == name {
			result = append(result, r)
		}
	}
	return result
}

func findByType(rs []rtypes.ResourceRecordSet, t rtypes.RRType) []rtypes.ResourceRecordSet {
	var result []rtypes.ResourceRecordSet
	for _, r := range rs {
		if r.Type == t {
			result = append(result, r)
		}
	}
	return result
}
