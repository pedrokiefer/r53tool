package dns

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/require"
)

func TestToAwsTags(t *testing.T) {
	in := []Tag{{Name: "env", Value: "prod"}, {Name: "team", Value: "dns"}}
	out := toAwsTags(in)
	require.Equal(t, []rtypes.Tag{
		{Key: aws.String("env"), Value: aws.String("prod")},
		{Key: aws.String("team"), Value: aws.String("dns")},
	}, out)
}
