package dns

import (
	"bytes"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func TestPrintResourceRecords_NoPanic(t *testing.T) {
	// Swap out stdout writer of tablewriter by capturing os.Stdout through buffer is not supported directly,
	// but the function should not panic with minimal input.
	records := []rtypes.ResourceRecordSet{
		{Name: aws.String("example.com."), Type: rtypes.RRTypeA},
	}
	// This test ensures it executes without crashing.
	// We cannot easily capture output without changing the API; keep it simple as requested.
	_ = bytes.NewBuffer(nil)
	PrintResourceRecords(records)
}
