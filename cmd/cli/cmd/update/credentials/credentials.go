// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/mesosphere/dynamic-credential-provider/cmd/cli/cmd/flags"
	"github.com/mesosphere/dynamic-credential-provider/pkg/credentialmanager/secret"
	"github.com/mesosphere/dynamic-credential-provider/pkg/k8s/client"
)

func NewCommand(cmdCfg *flags.CLIConfig) *cobra.Command {
	var (
		address  string
		username string
		password string
	)

	cmd := &cobra.Command{
		Use:   "registry-credentials [address] [username] [password]",
		Short: "Update image registry credentials",
		Long: `Update image registry credentials in the running cluster:

Examples:
  update registry-credentials --address=docker.io --username=myusername --password=mypassword
  update registry-credentials --address=myregistry:5000 --username=myusername --password=mypassword
  update registry-credentials --address=myregistry:5000/somepath --username=myusername --password=mypassword
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			k8sCLient, _, err := client.NewFromKubeconfig("")
			if err != nil {
				return err
			}

			manager := secret.NewSecretsCredentialManager(k8sCLient)

			err = manager.Update(context.Background(), address, username, password)
			if err != nil {
				return err
			}

			cmdCfg.Output.Infof("Updated credentials")
			return nil
		},
	}

	cmd.Flags().StringVar(&address, "address", "", "Address of the registry to update credentials")
	_ = cmd.MarkFlagRequired("address")
	cmd.Flags().StringVar(&username, "username", "", "New username for the registry")
	_ = cmd.MarkFlagRequired("username")
	cmd.Flags().StringVar(&password, "password", "", "New password for the registry")
	_ = cmd.MarkFlagRequired("password")

	return cmd
}
