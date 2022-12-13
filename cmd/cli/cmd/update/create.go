// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"github.com/spf13/cobra"

	"github.com/mesosphere/dynamic-credential-provider/cmd/cli/cmd/flags"
	"github.com/mesosphere/dynamic-credential-provider/cmd/cli/cmd/update/credentials"
)

func NewCommand(cmdCfg *flags.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update one of []",
	}

	cmd.AddCommand(credentials.NewCommand(cmdCfg))
	return cmd
}
