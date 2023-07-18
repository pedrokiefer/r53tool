package vuln

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/fatih/color"
	"github.com/pedrokiefer/route53copy/pkg/dig"
)

var WhiteBold = color.New(color.FgWhite, color.Bold)
var MISCONFIG = color.YellowString("[MISCONFIG]")
var VULN = color.RedString("[VULN]")

var cli = &http.Client{
	Timeout: 3 * time.Second,
}

type HTTPError struct {
	Reason string
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("http error: %s", e.Reason)
}

type domainCheck func(context.Context, *Findings, rtypes.ResourceRecordSet)

func checkError(err error, t, k string, name string, f *Findings, record rtypes.ResourceRecordSet) {
	var herr *HTTPError
	if errors.As(err, &herr) {
		switch herr.Reason {
		case "SSL not configured":
			f.MisconfigRecords = append(f.MisconfigRecords, MisConfigRRFromAWS(record, herr.Reason))
			log.Printf("%s Zone %s has %s %s to %s but SSL is not configured\n", MISCONFIG, f.Name, t, k, name)
		case "Invalid SSL certificate":
			f.MisconfigRecords = append(f.MisconfigRecords, MisConfigRRFromAWS(record, herr.Reason))
			log.Printf("%s Zone %s has %s %s to %s but the SSL certificate is invalid\n", MISCONFIG, f.Name, t, k, name)
		case "No such host":
			f.MisconfigRecords = append(f.MisconfigRecords, MisConfigRRFromAWS(record, herr.Reason))
			log.Printf("%s Zone %s has %s %s to %s but the distribution does not exist\n", MISCONFIG, f.Name, t, k, name)
		case "Forbidden":
			f.MisconfigRecords = append(f.MisconfigRecords, MisConfigRRFromAWS(record, herr.Reason))
			log.Printf("%s Zone %s has %s %s to %s S3 but the bucket is private\n", MISCONFIG, f.Name, t, k, name)
		default:
			log.Printf("%s error: %s\n", MISCONFIG, herr)
		}
	}
}

func valueInBody(b io.ReadCloser, v string) bool {
	defer b.Close()
	body, err := ioutil.ReadAll(b)
	if err != nil {
		return false
	}
	return strings.Contains(string(body), v)
}

func checkNoSuchBucket(ctx context.Context, name string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+name, nil)
	if err != nil {
		return false, err
	}
	resp, err := cli.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "tls: handshake failure") {
			log.Printf("SSL not configured for %s\n", name)
			return false, &HTTPError{Reason: "SSL not configured"}
		}
		if strings.Contains(err.Error(), "tls: failed to verify certificate") {
			log.Printf("Invalid SSL certificate for bucket %s\n", name)
			return false, &HTTPError{Reason: "Invalid SSL certificate"}
		}
		if strings.Contains(err.Error(), "no such host") {
			log.Printf("Bucket %s does not exist\n", name)
			return false, &HTTPError{Reason: "No such host"}
		}
		return false, nil
	}
	if resp.StatusCode == http.StatusForbidden {
		log.Printf("%s exists but is private\n", name)
		return false, &HTTPError{Reason: "Forbidden"}
	}
	return resp.StatusCode == http.StatusNotFound && valueInBody(resp.Body, "Code: NoSuchBucket"), nil
}

func checkAliasCloudFront(ctx context.Context, f *Findings, record rtypes.ResourceRecordSet) {
	if record.AliasTarget != nil && strings.HasSuffix(aws.ToString(record.AliasTarget.DNSName), ".cloudfront.net") && record.Type != rtypes.RRTypeAaaa {
		name := aws.ToString(record.Name)
		nok, err := checkNoSuchBucket(ctx, name)
		if err != nil {
			checkError(err, "an alias", "CloudFront", name, f, record)
		}
		if nok {
			f.VulnerableRecords = append(f.VulnerableRecords, RRFromAWS(record))
			log.Printf("%s Zone %s has an alias %s to CloudFront but the bucket does not exist\n", VULN, f.Name, name)
		}

	}
}

func checkCnameCloudFront(ctx context.Context, f *Findings, record rtypes.ResourceRecordSet) {
	if record.Type == rtypes.RRTypeCname && len(record.ResourceRecords) >= 1 && strings.HasSuffix(aws.ToString(record.ResourceRecords[0].Value), ".cloudfront.net") {
		name := aws.ToString(record.Name)
		nok, err := checkNoSuchBucket(ctx, name)
		if err != nil {
			checkError(err, "a CNAME", "CloudFront", name, f, record)
		}
		if nok {
			f.VulnerableRecords = append(f.VulnerableRecords, RRFromAWS(record))
			log.Printf("%s Zone %s has a CNAME %s to CloudFront but the bucket does not exist\n", VULN, f.Name, name)
		}
	}
}

func checkAliasElasticBeanStalk(ctx context.Context, f *Findings, record rtypes.ResourceRecordSet) {
	if record.AliasTarget != nil && strings.HasSuffix(aws.ToString(record.AliasTarget.DNSName), ".elasticbeanstalk.com") {
		name := aws.ToString(record.Name)
		dst := aws.ToString(record.AliasTarget.DNSName)
		err := dig.Resolve(ctx, name, "A")
		if err == nil {
			return
		}
		var derr *dig.ResolveError
		if errors.As(err, &derr) {
			if derr.Type == "NXDOMAIN" {
				f.VulnerableRecords = append(f.VulnerableRecords, RRFromAWS(record))
				log.Printf("%s Zone %s has an alias %s to Elastic Beanstalk %s but the domain does not exist\n", VULN, f.Name, name, dst)
			}
		}
	}
}

func checkCnameElasticBeanStalk(ctx context.Context, f *Findings, record rtypes.ResourceRecordSet) {
	if record.Type == rtypes.RRTypeCname && len(record.ResourceRecords) >= 1 && strings.HasSuffix(aws.ToString(record.ResourceRecords[0].Value), ".elasticbeanstalk.com") {
		name := aws.ToString(record.Name)
		dst := aws.ToString(record.ResourceRecords[0].Value)
		err := dig.Resolve(ctx, name, "A")
		if err == nil {
			return
		}
		var derr *dig.ResolveError
		if errors.As(err, &derr) {
			if derr.Type == "NXDOMAIN" {
				cerr := dig.Resolve(ctx, name, "CNAME")
				if cerr == nil {
					f.VulnerableRecords = append(f.VulnerableRecords, RRFromAWS(record))
					log.Printf("%s Zone %s has a CNAME %s to Elastic Beanstalk %s but the domain does not exist\n", VULN, f.Name, name, dst)
					return
				}
			}
		}
	}
}

func checkCnameS3(ctx context.Context, f *Findings, record rtypes.ResourceRecordSet) {
	if record.Type != rtypes.RRTypeCname || len(record.ResourceRecords) < 1 {
		return
	}
	dst := aws.ToString(record.ResourceRecords[0].Value)
	if (strings.HasSuffix(dst, "amazonaws.com") && strings.Contains(dst, "s3")) ||
		strings.Contains(dst, ".s3-website") {
		name := aws.ToString(record.Name)
		nok, err := checkNoSuchBucket(ctx, dst)
		if err != nil {
			checkError(err, "a CNAME", "S3", name, f, record)
		}
		if nok {
			f.VulnerableRecords = append(f.VulnerableRecords, RRFromAWS(record))
			log.Printf("%s Zone %s has a CNAME %s to S3 but the bucket does not exist\n", VULN, f.Name, name)
		}
	}
}

func checkAliasS3(ctx context.Context, f *Findings, record rtypes.ResourceRecordSet) {
	if record.AliasTarget == nil {
		return
	}
	dst := aws.ToString(record.AliasTarget.DNSName)
	if (strings.HasSuffix(dst, "amazonaws.com") && strings.Contains(dst, "s3")) ||
		strings.Contains(dst, ".s3-website") {
		name := aws.ToString(record.Name)
		nok, err := checkNoSuchBucket(ctx, dst)
		if err != nil {
			checkError(err, "an alias", "S3", name, f, record)
		}
		if nok {
			f.VulnerableRecords = append(f.VulnerableRecords, RRFromAWS(record))
			log.Printf("%s Zone %s has an alias %s to S3 but the bucket does not exist\n", VULN, f.Name, name)
		}
	}
}

var subDomainTakeoverChecks = []domainCheck{
	checkAliasCloudFront,
	checkCnameCloudFront,
	checkAliasElasticBeanStalk,
	checkCnameElasticBeanStalk,
	checkCnameS3,
	checkAliasS3,
	checkCNameExists,
	checkAliasExists,
}

func SubDomainTakeoverCheck(ctx context.Context, f *Findings, record rtypes.ResourceRecordSet) {
	for _, check := range subDomainTakeoverChecks {
		check(ctx, f, record)
	}
}
