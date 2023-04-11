package vuln

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func CheckSPF(name string, rs []rtypes.ResourceRecordSet) []string {
	hasSPF := 0
	issues := []string{}
	txt := findByTypeAndName(rs, rtypes.RRTypeTxt, name)
	if len(txt) == 0 {
		issues = append(issues, fmt.Sprintf("%s %s has no TXT record\n", MISCONFIG, name))
		return issues
	}

	for _, t := range txt {
		for _, v := range t.ResourceRecords {
			value := aws.ToString(v.Value)
			if !strings.Contains(value, "v=spf1") {
				continue
			}
			hasSPF++
			issues = append(issues, spfScan(value)...)
		}
	}

	if hasSPF == 0 {
		issues = append([]string{fmt.Sprintf("%s %s MX is missing SPF record\n", MISCONFIG, name)}, issues...)
	} else if hasSPF > 1 {
		issues = append([]string{fmt.Sprintf("%s %s has multiple SPF records\n", MISCONFIG, name)}, issues...)
	}
	return issues
}

func spfScan(spf string) []string {
	return []string{}
}

// 'SPF_NO_ALL': {
// 	'code': 2,
// 	'title': 'No \'all\' mechanism',
// 	'detail': 'There is no all mechanism in the record. It may be possible'
// 			  ' to spoof the domain without causing an SPF failure.'
// },
// 'SPF_PASS_ALL': {
// 	'code': 3,
// 	'title': '\'Pass\' qualifer for \'all\' mechanism',
// 	'detail': 'The \'all\' mechanism uses the \'Pass\' qualifer \'+\'. '
// 			  'It should be possible to spoof the domain without causing '
// 			  'an SPF failure.'
// },
// 'SPF_SOFT_FAIL_ALL': {
// 	'code': 4,
// 	'title': '\'SoftFail\' qualifer for \'all\' mechanism',
// 	'detail': 'The \'all\' mechanism uses the \'SoftFail\' qualifer \'~\'.'
// 			  ' It should be possible to spoof the domain by only causing '
// 			  'a soft SPF failure. Most filters will let this through by '
// 			  'only raising the total spam score.'
// },
// 'SPF_LOOKUP_ERROR': {
// 	'code': 5,
// 	'title': 'Too many lookups for SPF validation',
// 	'detail': 'The SPF record requires more than 10 DNS lookups for the '
// 			  'validation process. The RFC states that maximum 10 lookups '
// 			  'are allowed. As a result, recipients may throw a PermError '
// 			  'instead of proceeding with SPF validation. Recipients will '
// 			  'treat these errors differently than a hard or soft SPF fail'
// 			  ' , and some will continue processing the mail.'
// },
// 'SPF_UNREGISTERED_DOMAINS': {
// 	'code': 6,
// 	'title': 'Unregistered domains in SPF validation chain',
// 	'detail': 'One or more domains used in the SPF validation process are '
// 			  'presently unregistered. An attacker could register these '
// 			  'and configure his own SPF record to be included in the '
// 			  'validation logic. The affected domains are: {domains}'
// },
// 'SPF_RECURSE': {
// 	'code': 12,
// 	'title': 'Trivial SPF recurse',
// 	'detail': 'Infinite recurse loop with the domain {recursive_domain} '
// 			  'included in the validation chain for {domain}'
// }
