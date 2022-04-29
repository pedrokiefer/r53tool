package cli

import "github.com/spf13/cobra"

func NewRunner(version string) *cobra.Command {
	versionStr = version
	return rootCmd
}
