package dns

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type ParkedResources struct {
	HasARecord        bool
	HasAAAARecord     bool
	HasWWWCnameRecord bool
}

func RemoveResourceRecordsWithTypes(records []rtypes.ResourceRecordSet, types []rtypes.RRType) []rtypes.ResourceRecordSet {
	filtered := []rtypes.ResourceRecordSet{}
	for _, record := range records {
		if !typeInList(types, record.Type) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func FindParkedResourceRecord(records []rtypes.ResourceRecordSet, fqdn string) ([]rtypes.ResourceRecordSet, ParkedResources) {
	rr := []rtypes.ResourceRecordSet{}
	www := fmt.Sprintf("www.%s", fqdn)
	p := ParkedResources{}
	for _, record := range records {
		if record.Type == rtypes.RRTypeA && aws.ToString(record.Name) == fqdn {
			p.HasARecord = true
			rr = append(rr, record)
			continue
		}
		if record.Type == rtypes.RRTypeAaaa && aws.ToString(record.Name) == fqdn {
			p.HasAAAARecord = true
			rr = append(rr, record)
			continue
		}
		if record.Type == rtypes.RRTypeCname && aws.ToString(record.Name) == www {
			p.HasWWWCnameRecord = true
			rr = append(rr, record)
			continue
		}
	}
	return rr, p
}

func FindNSRecord(records []rtypes.ResourceRecordSet) (rtypes.ResourceRecordSet, error) {
	for _, record := range records {
		if record.Type == rtypes.RRTypeNs {
			return record, nil
		}
	}
	return rtypes.ResourceRecordSet{}, fmt.Errorf("no NS record found")
}

func typeInList(types []rtypes.RRType, t rtypes.RRType) bool {
	for _, t2 := range types {
		if t == t2 {
			return true
		}
	}
	return false
}
