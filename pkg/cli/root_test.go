package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRootCmdInit(t *testing.T) {
	c := newRootCmd()
	require.IsType(t, &cobra.Command{}, c)
	// Flags exist
	f := c.PersistentFlags()
	require.NotNil(t, f.Lookup("dry"))
	require.NotNil(t, f.Lookup("no-wait"))
}
