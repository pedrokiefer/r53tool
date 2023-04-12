package vuln

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type ZoneMeta struct {
	ZoneID string `json:"zone_id,omitempty"`
	Name   string `json:"name,omitempty"`
}

type ResourceRecord struct {
	Name   string   `json:"name,omitempty"`
	Type   string   `json:"type,omitempty"`
	Alias  string   `json:"alias,omitempty"`
	Values []string `json:"values,omitempty"`
}

type MisConfigResourceRecord struct {
	ResourceRecord
	Reason string `json:"reason,omitempty"`
}

type Findings struct {
	ZoneID            string                    `json:"zone_id,omitempty"`
	Name              string                    `json:"name,omitempty"`
	VulnerableRecords []ResourceRecord          `json:"vulnerable_records,omitempty"`
	MisconfigRecords  []MisConfigResourceRecord `json:"misconfig_records,omitempty"`
}

func NewFindings(zm ZoneMeta) *Findings {
	return &Findings{
		ZoneID:            zm.ZoneID,
		Name:              zm.Name,
		VulnerableRecords: []ResourceRecord{},
		MisconfigRecords:  []MisConfigResourceRecord{},
	}
}

func RRFromAWS(awsRR rtypes.ResourceRecordSet) ResourceRecord {
	rr := ResourceRecord{
		Name: aws.ToString(awsRR.Name),
		Type: string(awsRR.Type),
	}
	if awsRR.AliasTarget != nil {
		rr.Alias = aws.ToString(awsRR.AliasTarget.DNSName)
	}
	for _, v := range awsRR.ResourceRecords {
		rr.Values = append(rr.Values, aws.ToString(v.Value))
	}
	return rr
}

func MisConfigRRFromAWS(awsRR rtypes.ResourceRecordSet, reason string) MisConfigResourceRecord {
	rr := MisConfigResourceRecord{
		ResourceRecord: RRFromAWS(awsRR),
		Reason:         reason,
	}
	return rr
}
