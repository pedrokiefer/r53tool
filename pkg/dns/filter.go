package dns

import (
	"fmt"

	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func RemoveResourceRecordsWithTypes(records []rtypes.ResourceRecordSet, types []rtypes.RRType) []rtypes.ResourceRecordSet {
	filtered := []rtypes.ResourceRecordSet{}
	for _, record := range records {
		if !typeInList(types, record.Type) {
			filtered = append(filtered, record)
		}
	}
	return filtered
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
