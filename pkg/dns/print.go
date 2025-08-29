package dns

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/olekukonko/tablewriter"
)

func PrintResourceRecords(records []rtypes.ResourceRecordSet) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Name", "Type", "Value"})

	for _, record := range records {
		values := ""
		for i, v := range record.ResourceRecords {
			values += aws.ToString(v.Value)
			if i != len(record.ResourceRecords)-1 {
				values += "\n"
			}
		}
		table.Append([]string{aws.ToString(record.Name), string(record.Type), values})
	}

	table.Render()
}
