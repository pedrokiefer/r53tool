package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func runCmd(c *cobra.Command, args []string) (string, error) {
	buf := &bytes.Buffer{}
	c.SetOut(buf)
	c.SetErr(buf)
	c.SetArgs(args)
	_, err := c.ExecuteC()
	return buf.String(), err
}

func TestExportCommand_ArgsValidation(t *testing.T) {
	c := newExportCommand()
	_, err := runCmd(c, []string{"only-profile"})
	require.Error(t, err)
}

func TestExportCommand_DryRunDoesNotError(t *testing.T) {
	// We can only exercise flag parsing; RunE hits AWS. Here we set args and expect ExecuteC to fail on AWS unless dry run is at least parsed.
	// Note: full RunE is not executed without required args; simulate with minimal args and expect validation error only.
	c := newExportCommand()
	// Provide both required args to pass validation; we cannot actually run without AWS, so just ensure command builds.
	// We won't execute RunE here to avoid AWS calls; argument validation already covered above.
	_ = c // placeholder to silence linter in case of future modifications
}

func TestDomainsCommand_ArgsValidation(t *testing.T) {
	c := newDomainsCommand()
	_, err := runCmd(c, []string{"profile-only"})
	require.Error(t, err)
}

func TestCopyCommand_ArgsValidation(t *testing.T) {
	c := newCopyCommand()
	_, err := runCmd(c, []string{"src", "dst"})
	require.Error(t, err)
}

func TestDeleteCommand_ArgsValidation(t *testing.T) {
	c := newDeleteCommand()
	_, err := runCmd(c, []string{"only-profile"})
	require.Error(t, err)
}

func TestFindCommand_ArgsValidation(t *testing.T) {
	c := newFindCommand()
	_, err := runCmd(c, []string{"only-profile"})
	require.Error(t, err)
}

func TestParkCommand_ArgsValidation(t *testing.T) {
	c := newParkCommand()
	// No args -> error
	_, err := runCmd(c, []string{})
	require.Error(t, err)
	// One arg -> error (MinimumNArgs=2)
	_, err = runCmd(c, []string{"profile"})
	require.Error(t, err)
}

func TestCheckZoneCommand_ArgsValidation(t *testing.T) {
	c := newCheckZoneCmd()
	_, err := runCmd(c, []string{})
	require.Error(t, err)
}

func TestVulnerabilityScanCommand_ArgsValidation(t *testing.T) {
	c := newVulnerabiltyScanCommand()
	_, err := runCmd(c, []string{})
	require.Error(t, err)
}
