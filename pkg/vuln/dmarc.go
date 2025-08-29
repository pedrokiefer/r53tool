package vuln

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func CheckDMARC(name string, rs []rtypes.ResourceRecordSet) []string {
	hasDMARC := 0
	issues := []string{}
	txt := findByTypeAndName(rs, rtypes.RRTypeTxt, fmt.Sprintf("_dmarc.%s", name))
	for _, t := range txt {
		for _, v := range t.ResourceRecords {
			if !strings.Contains(aws.ToString(v.Value), "v=DMARC1;") {
				continue
			}
			hasDMARC++
			dmarc := aws.ToString(v.Value)
			issues = append(issues, dmarcScan(dmarc)...)
		}
	}
	if hasDMARC == 0 {
		issues = append([]string{fmt.Sprintf("%s %s is a MX with no DMARC record\n", MISCONFIG, name)}, issues...)
	} else if hasDMARC > 1 {
		issues = append([]string{fmt.Sprintf("%s %s has multiple DMARC records\n", MISCONFIG, name)}, issues...)
	}
	return issues
}

func dmarcScan(dmarc string) []string {
	var result []string
	terms := strings.Split(dmarc, ";")
	for _, term := range terms {
		term = strings.TrimSpace(term)
		parts := strings.Split(term, "=")
		if len(parts) != 2 {
			continue
		}

		tag := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch tag {
		case "p":
			if value != "reject" && value != "quarantine" {
				result = append(result, fmt.Sprintf("%s DMARC policy is %s, which allows spoofed emails\n", VULN, value))
			}
		case "sp":
			if value != "reject" && value != "quarantine" {
				result = append(result, fmt.Sprintf("%s DMARC subdomain policy is %s, which allows spoofed emails\n", VULN, value))
			}
		case "pct":
			v, err := strconv.Atoi(value)
			if err != nil {
				result = append(result, fmt.Sprintf("%s DMARC policy pct has invalid value: %s\n", VULN, value))
				continue
			}
			if v < 100 {
				result = append(result, fmt.Sprintf("%s DMARC policy is only applied to %d%% of emails\n", VULN, v))
			} else if v > 100 {
				result = append(result, fmt.Sprintf("%s DMARC policy pct has invalid value: %s\n", VULN, value))
			}
		}
	}
	return result
}
