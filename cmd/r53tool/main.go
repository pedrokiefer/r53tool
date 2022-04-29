package main

import (
	"fmt"

	"github.com/pedrokiefer/route53copy/cmd"
	"github.com/pedrokiefer/route53copy/pkg/cli"
)

func main() {
	version := fmt.Sprintf("%s, commit: %s, built: %s", cmd.Version, cmd.Commit, cmd.BuildDate)
	runner := cli.NewRunner(version)
	cmd.Run(runner)
}
