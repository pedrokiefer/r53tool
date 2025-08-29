package dns

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/StackExchange/dnscontrol/v4/pkg/prettyzone"
	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// WriteBindZoneFile writes a BIND 9-compatible zone file at outputPath for the given zone and record sets.
// - zone should be the zone/apex name (with or without trailing dot). It will be normalized automatically.
// - records are the Route53 ResourceRecordSets to export.
// Unsupported AWS-specific constructs (like AliasTarget) are skipped and added as comments at the top.
func WriteBindZoneFile(outputPath string, zone string, records []rtypes.ResourceRecordSet) error {
	origin := DenormalizeDomain(zone) // origin without trailing dot

	recs, comments := route53ToModelRecords(origin, records)

	// Compute a reasonable default TTL (most common across records, excluding NS per dnscontrol's logic)
	defaultTTL := prettyzone.MostCommonTTL(recs)

	// Ensure stable/pretty order (optional but produces nicer output)
	_ = prettyzone.PrettySort(recs, origin, defaultTTL, comments)

	// Create/overwrite the output file
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write the zone file
	if err := prettyzone.WriteZoneFileRC(f, recs, origin, defaultTTL, comments); err != nil {
		return err
	}
	return nil
}

func route53ToModelRecords(origin string, rs []rtypes.ResourceRecordSet) (models.Records, []string) {
	records := models.Records{}
	comments := []string{fmt.Sprintf("; Exported by r53tool. Zone: %s", origin)}
	skippedAlias := []string{}

	for _, rrset := range rs {
		// Skip AWS AliasTarget records (not representable in BIND zone files in many cases)
		if rrset.AliasTarget != nil {
			skippedAlias = append(skippedAlias, fmt.Sprintf("%s %s -> %s", aws.ToString(rrset.Name), rrset.Type, aws.ToString(rrset.AliasTarget.DNSName)))
			continue
		}

		rtype := string(rrset.Type)
		nameFQDN := aws.ToString(rrset.Name)
		ttl := uint32(0)
		if rrset.TTL != nil {
			ttl = uint32(aws.ToInt64(rrset.TTL))
		}

		// Some types have a single value string with combined RDATA. Others are one-per-value.
		if len(rrset.ResourceRecords) == 0 {
			// Some providers/types might have empty ResourceRecords (unlikely without AliasTarget).
			continue
		}

		for _, v := range rrset.ResourceRecords {
			val := strings.TrimSpace(aws.ToString(v.Value))

			rc := &models.RecordConfig{Type: rtype, TTL: ttl}
			// Set label from FQDN (dnscontrol expects origin without trailing dot)
			rc.SetLabelFromFQDN(nameFQDN, origin)

			switch rtype {
			case "A", "AAAA":
				// For A/AAAA, each value is an IP. Use SetTarget to let dnscontrol validate.
				_ = rc.SetTarget(val)

			case "CNAME", "NS", "PTR":
				// Targets may come quoted or with trailing dots. Let CanonicalizeTargets handle FQDNs later.
				_ = rc.SetTarget(models.StripQuotes(val))

			case "MX":
				// e.g. "10 mail.example.com." or "10 mail.example.com"
				_ = rc.SetTargetMXString(models.StripQuotes(val))

			case "SRV":
				// e.g. "10 5 8080 target.example.com."
				_ = rc.SetTargetSRVString(models.StripQuotes(val))

			case "CAA":
				// e.g. "0 issue \"letsencrypt.org\""
				_ = rc.SetTargetCAAString(models.StripQuotes(val))

			case "SOA":
				// Single value string with all SOA fields
				_ = rc.SetTargetSOAString(models.StripQuotes(val))

			case "TXT", "SPF":
				// TXT can be segmented. AWS typically returns quoted strings possibly with spaces/escapes.
				parts := models.ParseQuotedTxt(val)
				if len(parts) <= 1 {
					// Either not quoted or a single segment
					_ = rc.SetTargetTXT(models.StripQuotes(val))
				} else {
					_ = rc.SetTargetTXTs(parts)
				}

			default:
				// Fallback: try generic population parser for RFC1035-like rdata
				_ = rc.PopulateFromString(rtype, models.StripQuotes(val), origin)
			}

			records = append(records, rc)
		}
	}

	// Post-processing: canonicalize targets (turn relative into FQDNs)
	models.CanonicalizeTargets(records, origin)

	// Add comments for skipped alias records (sorted for stability)
	if len(skippedAlias) > 0 {
		sort.Strings(skippedAlias)
		comments = append(comments, "; NOTE: The following AWS Alias records were skipped (not supported in BIND):")
		for _, s := range skippedAlias {
			comments = append(comments, ";   "+s)
		}
	}

	return records, comments
}
