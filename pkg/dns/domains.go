package dns

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53domains"
	"github.com/aws/aws-sdk-go-v2/service/route53domains/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/middleware"
	smithytime "github.com/aws/smithy-go/time"
	smithywaiter "github.com/aws/smithy-go/waiter"
)

type DomainManager struct {
	cli    *route53domains.Client
	stscli *sts.Client
}

type Transfer struct {
	Password    string
	OperationID string
}

func NewDomainManager(ctx context.Context, profile string) (*DomainManager, error) {
	if r := os.Getenv("AWS_REGION"); r == "" {
		os.Setenv("AWS_REGION", "us-east-1")
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile))

	if err != nil {
		return nil, err
	}

	return &DomainManager{
		cli:    route53domains.NewFromConfig(cfg),
		stscli: sts.NewFromConfig(cfg),
	}, nil
}

func (dm *DomainManager) GetAccountID(ctx context.Context) (string, error) {
	i, err := dm.stscli.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return aws.ToString(i.Account), nil
}

func (dm *DomainManager) ListRegisteredDomains(ctx context.Context) ([]string, error) {
	paginator := route53domains.NewListDomainsPaginator(dm.cli, &route53domains.ListDomainsInput{})

	domains := []string{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, domain := range page.Domains {
			domains = append(domains, aws.ToString(domain.DomainName))
		}
	}

	return domains, nil
}

func (dm *DomainManager) TransferDomain(ctx context.Context, domain, dstAccount string) (*Transfer, error) {
	resp, err := dm.cli.TransferDomainToAnotherAwsAccount(ctx, &route53domains.TransferDomainToAnotherAwsAccountInput{
		AccountId:  aws.String(dstAccount),
		DomainName: aws.String(domain),
	})
	if err != nil {
		return nil, err
	}

	return &Transfer{
		Password:    aws.ToString(resp.Password),
		OperationID: aws.ToString(resp.OperationId),
	}, nil
}

func (dm *DomainManager) WaitOperation(ctx context.Context, expected types.OperationStatus, opID string, d time.Duration) error {
	w := NewGetOperationDetailWaiter(dm.cli, func(godwo *GetOperationDetailWaiterOptions) {
		godwo.ExpectedStatus = expected
	})
	return w.Wait(ctx, &route53domains.GetOperationDetailInput{
		OperationId: aws.String(opID),
	}, d)
}

func (dm *DomainManager) CancelTranfer(ctx context.Context, domain string) (string, error) {
	resp, err := dm.cli.CancelDomainTransferToAnotherAwsAccount(ctx, &route53domains.CancelDomainTransferToAnotherAwsAccountInput{
		DomainName: aws.String(domain),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.OperationId), nil
}

func (dm *DomainManager) AcceptTransfer(ctx context.Context, domain, password string) (string, error) {
	resp, err := dm.cli.AcceptDomainTransferFromAnotherAwsAccount(ctx, &route53domains.AcceptDomainTransferFromAnotherAwsAccountInput{
		DomainName: aws.String(domain),
		Password:   aws.String(password),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.OperationId), nil
}

// GetOperationDetailAPIClient is a client that implements the GetOperationDetail operation.
type GetOperationDetailAPIClient interface {
	GetOperationDetail(ctx context.Context, params *route53domains.GetOperationDetailInput, optFns ...func(*route53domains.Options)) (*route53domains.GetOperationDetailOutput, error)
}

var _ GetOperationDetailAPIClient = (*route53domains.Client)(nil)

// GetOperationDetailWaiterOptions are waiter options for
// GetOperationDetailWaiter
type GetOperationDetailWaiterOptions struct {

	// Set of options to modify how an operation is invoked. These apply to all
	// operations invoked for this client. Use functional options on operation call to
	// modify this list for per operation behavior.
	APIOptions []func(*middleware.Stack) error

	// MinDelay is the minimum amount of time to delay between retries. If unset,
	// GetOperationDetailWaiter will use default minimum delay of 30 seconds.
	// Note that MinDelay must resolve to a value lesser than or equal to the MaxDelay.
	MinDelay time.Duration

	// MaxDelay is the maximum amount of time to delay between retries. If unset or set
	// to zero, GetOperationDetailWaiter will use default max delay of 120
	// seconds. Note that MaxDelay must resolve to value greater than or equal to the
	// MinDelay.
	MaxDelay time.Duration

	// ExpectedStatus is the expected status of the operation. If unset,
	// GetOperationDetailWaiter will use default expected status of types.OperationStatusSuccessful.
	ExpectedStatus types.OperationStatus

	// LogWaitAttempts is used to enable logging for waiter retry attempts
	LogWaitAttempts bool

	// Retryable is function that can be used to override the service defined
	// waiter-behavior based on operation output, or returned error. This function is
	// used by the waiter to decide if a state is retryable or a terminal state. By
	// default service-modeled logic will populate this option. This option can thus be
	// used to define a custom waiter state with fall-back to service-modeled waiter
	// state mutators.The function returns an error in case of a failure state. In case
	// of retry state, this function returns a bool value of true and nil error, while
	// in case of success it returns a bool value of false and nil error.
	Retryable func(context.Context, types.OperationStatus, *route53domains.GetOperationDetailInput, *route53domains.GetOperationDetailOutput, error) (bool, error)
}

// GetOperationDetailWaiter defines the waiters for
// GetOperationDetail
type GetOperationDetailWaiter struct {
	client GetOperationDetailAPIClient

	options GetOperationDetailWaiterOptions
}

// NewGetOperationDetailWaiter constructs a GetOperationDetailWaiter.
func NewGetOperationDetailWaiter(client GetOperationDetailAPIClient, optFns ...func(*GetOperationDetailWaiterOptions)) *GetOperationDetailWaiter {
	options := GetOperationDetailWaiterOptions{}
	options.MinDelay = 30 * time.Second
	options.MaxDelay = 120 * time.Second
	options.Retryable = getOperationDetailStateRetryable
	options.ExpectedStatus = types.OperationStatusSuccessful

	for _, fn := range optFns {
		fn(&options)
	}
	return &GetOperationDetailWaiter{
		client:  client,
		options: options,
	}
}

// Wait calls the waiter function for GetOperationDetail waiter. The
// maxWaitDur is the maximum wait duration the waiter will wait. The maxWaitDur is
// required and must be greater than zero.
func (w *GetOperationDetailWaiter) Wait(ctx context.Context, params *route53domains.GetOperationDetailInput, maxWaitDur time.Duration, optFns ...func(*GetOperationDetailWaiterOptions)) error {
	_, err := w.WaitForOutput(ctx, params, maxWaitDur, optFns...)
	return err
}

// WaitForOutput calls the waiter function for GetOperationDetail waiter and
// returns the output of the successful operation. The maxWaitDur is the maximum
// wait duration the waiter will wait. The maxWaitDur is required and must be
// greater than zero.
func (w *GetOperationDetailWaiter) WaitForOutput(ctx context.Context, params *route53domains.GetOperationDetailInput, maxWaitDur time.Duration, optFns ...func(*GetOperationDetailWaiterOptions)) (*route53domains.GetOperationDetailOutput, error) {
	if maxWaitDur <= 0 {
		return nil, fmt.Errorf("maximum wait time for waiter must be greater than zero")
	}

	options := w.options
	for _, fn := range optFns {
		fn(&options)
	}

	if options.MaxDelay <= 0 {
		options.MaxDelay = 120 * time.Second
	}

	if options.MinDelay > options.MaxDelay {
		return nil, fmt.Errorf("minimum waiter delay %v must be lesser than or equal to maximum waiter delay of %v.", options.MinDelay, options.MaxDelay)
	}

	ctx, cancelFn := context.WithTimeout(ctx, maxWaitDur)
	defer cancelFn()

	logger := smithywaiter.Logger{}
	remainingTime := maxWaitDur

	var attempt int64
	for {

		attempt++
		apiOptions := options.APIOptions
		start := time.Now()

		if options.LogWaitAttempts {
			logger.Attempt = attempt
			apiOptions = append([]func(*middleware.Stack) error{}, options.APIOptions...)
			apiOptions = append(apiOptions, logger.AddLogger)
		}

		out, err := w.client.GetOperationDetail(ctx, params, func(o *route53domains.Options) {
			o.APIOptions = append(o.APIOptions, apiOptions...)
		})

		retryable, err := options.Retryable(ctx, w.options.ExpectedStatus, params, out, err)
		if err != nil {
			return nil, err
		}
		if !retryable {
			return out, nil
		}

		remainingTime -= time.Since(start)
		if remainingTime < options.MinDelay || remainingTime <= 0 {
			break
		}

		// compute exponential backoff between waiter retries
		delay, err := smithywaiter.ComputeDelay(
			attempt, options.MinDelay, options.MaxDelay, remainingTime,
		)
		if err != nil {
			return nil, fmt.Errorf("error computing waiter delay, %w", err)
		}

		remainingTime -= delay
		// sleep for the delay amount before invoking a request
		if err := smithytime.SleepWithContext(ctx, delay); err != nil {
			return nil, fmt.Errorf("request cancelled while waiting, %w", err)
		}
	}
	return nil, fmt.Errorf("exceeded max wait time for GetOperationDetail waiter")
}

func getOperationDetailStateRetryable(ctx context.Context, expected types.OperationStatus, input *route53domains.GetOperationDetailInput, output *route53domains.GetOperationDetailOutput, err error) (bool, error) {

	if err == nil {
		if output.Status == expected {
			return false, nil
		}

		if output.Status == types.OperationStatusFailed || output.Status == types.OperationStatusError {
			return false, fmt.Errorf("operation failed: %s", aws.ToString(output.Message))
		}
	}

	return true, nil
}
