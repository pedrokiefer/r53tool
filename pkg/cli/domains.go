package cli

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/route53domains/types"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/spf13/cobra"
)

type domainsApp struct {
	SourceProfile      string
	DestinationProfile string
}

func init() {
	rootCmd.AddCommand(newDomainsCommand())
}

func (a *domainsApp) Run(ctx context.Context) error {
	srcManager, err := dns.NewDomainManager(ctx, a.SourceProfile)
	if err != nil {
		return err
	}

	dstManager, err := dns.NewDomainManager(ctx, a.DestinationProfile)
	if err != nil {
		return err
	}

	accountID, err := dstManager.GetAccountID(ctx)
	if err != nil {
		return err
	}

	domains, err := srcManager.ListRegisteredDomains(ctx)
	if err != nil {
		return err
	}

	log.Printf("Found %d domains in %s to transfer\n", len(domains), a.SourceProfile)

	if dryRun {
		log.Printf("Dry run... \n The following domains will be copied: \n")
		log.Println(domains)
		return nil
	}

	for _, domain := range domains {
		log.Printf("Transferring domain %s...\n", domain)
		t, err := srcManager.TransferDomain(ctx, domain, accountID)
		if err != nil {
			continue
		}

		err = srcManager.WaitOperation(ctx, types.OperationStatusInProgress, t.OperationID, 5*time.Minute)
		if err != nil {
			return err
		}

		log.Print("Waiting some more...")
		time.Sleep(30 * time.Second)

		log.Printf("Domain transfer initiated for %s: %s\n", domain, t.OperationID)
		opID, err := dstManager.AcceptTransfer(ctx, domain, t.Password)
		if err != nil {
			log.Printf("failed to accept transfer for %s: %+v", domain, err)
			copID, cerr := srcManager.CancelTranfer(ctx, domain)
			if cerr != nil {
				return fmt.Errorf("failed to cancel transfer for %s: %s", domain, cerr)
			}
			log.Printf("cancelled transfer for %s: %s", domain, copID)
			return err
		}

		err = dstManager.WaitOperation(ctx, types.OperationStatusSuccessful, opID, 5*time.Minute)
		if err != nil {
			return err
		}

		log.Printf("Domain transfer accepted for %s: %s\n", domain, opID)
	}

	return nil
}

func newDomainsCommand() *cobra.Command {
	a := domainsApp{}

	c := &cobra.Command{
		Use:   "domains <source_profile> <dest_profile>",
		Short: "Domains is a tool to move registered domains from one AWS account to another",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.SourceProfile = args[0]
			a.DestinationProfile = args[1]
			return a.Run(cmd.Context())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	return c
}
