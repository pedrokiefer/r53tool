package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// captureStdout runs f while capturing writes to os.Stdout and returns the output string.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	done := make(chan struct{})
	var out []byte
	go func() {
		defer close(done)
		var readErr error
		out, readErr = io.ReadAll(r)
		if readErr != nil {
			out = []byte{}
		}
	}()
	f()
	_ = w.Close()
	<-done
	os.Stdout = old
	return string(out)
}

func TestVersionCommand_PrintsVersion(t *testing.T) {
	cmd := NewRunner("1.2.3")
	cmd.SetArgs([]string{"version"})

	out := captureStdout(t, func() {
		_, err := cmd.ExecuteC()
		require.NoError(t, err)
	})

	require.True(t, strings.Contains(out, "r53tool version 1.2.3"), out)
}
