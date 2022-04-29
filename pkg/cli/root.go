package cli

import "github.com/spf13/cobra"

var (
	//flags
	dryRun bool

	rootCmd = newRootCmd()
)

func newRootCmd() *cobra.Command {
	c := &cobra.Command{
		Use:           "r53tool",
		Short:         "r53tool is a swiss army knife for Route53",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	f := c.PersistentFlags()
	f.BoolVar(&dryRun, "dry", false, "Dry run")
	return c
}
