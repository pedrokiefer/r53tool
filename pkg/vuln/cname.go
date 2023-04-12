package vuln

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dig"
)

func checkCNameExists(ctx context.Context, f *Findings, rs rtypes.ResourceRecordSet) {
	if rs.Type != rtypes.RRTypeCname {
		return
	}
	if len(rs.ResourceRecords) == 0 {
		return
	}

	name := aws.ToString(rs.Name)
	dst := aws.ToString(rs.ResourceRecords[0].Value)
	err := dig.Resolve(ctx, name, "A")
	if err == nil {
		return
	}

	var derr *dig.ResolveError
	if errors.As(err, &derr) {
		if derr.Type == "NXDOMAIN" {
			f.MisconfigRecords = append(f.MisconfigRecords, MisConfigRRFromAWS(rs, "CNAME points to missing name"))
			log.Printf("%s CNAME %s points to missing name %s\n", MISCONFIG, name, dst)
		}
	}
}
