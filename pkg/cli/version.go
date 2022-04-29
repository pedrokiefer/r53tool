package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionStr = "0.0.1"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of r53tool",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("r53tool version %\n", versionStr)
	},
}
