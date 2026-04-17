package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var _ = cobra.Command{}
