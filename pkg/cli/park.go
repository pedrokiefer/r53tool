package cli

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

type parkApp struct {
	Profile       string
	DestinationIP string
}

func init() {
	rootCmd.AddCommand(newParkCommand())
}

func (a *parkApp) Run(ctx context.Context) error {

	log.Printf("Parking domains in %s...\n", a.Profile)

	return nil
}

func newParkCommand() *cobra.Command {
	a := parkApp{}

	c := &cobra.Command{
		Use:   "park <profile> <destination_ip>",
		Short: "Park is a tool to park domains in Route53 creating A and www CNAME records",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Profile = args[0]
			a.DestinationIP = args[1]
			return a.Run(cmd.Context())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	return c
}
