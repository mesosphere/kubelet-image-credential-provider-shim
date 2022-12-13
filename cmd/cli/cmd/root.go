// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mesosphere/dkp-cli-runtime/core/cmd/root"

	"github.com/mesosphere/dynamic-credential-provider/cmd/cli/cmd/flags"
	"github.com/mesosphere/dynamic-credential-provider/cmd/cli/cmd/update"
)

func NewCommand(in io.Reader, out, errOut io.Writer) (*cobra.Command, *flags.CLIConfig) {
	rootCmd, rootOptions := root.NewCommand(out, errOut)
	rootCmd.Use = "credential-manager"
	rootCmd.Short = "Create and dynamically manage registry credentials"
	rootCmd.SilenceUsage = true
	// disable cobra built-in error printing, we output the error with formatting.
	rootCmd.SilenceErrors = true
	rootCmd.DisableAutoGenTag = true

	config := &flags.CLIConfig{
		In:     in,
		Output: rootOptions.Output,
	}

	rootCmd.AddCommand(update.NewCommand(config))

	return rootCmd, config
}

func Execute() {
	rootCmd, config := NewCommand(os.Stdin, os.Stdout, os.Stderr)

	if err := rootCmd.Execute(); err != nil {
		config.Output.Error(err, "")
		//nolint:revive // Common to do this in Cobra
		os.Exit(1)
	}
}
