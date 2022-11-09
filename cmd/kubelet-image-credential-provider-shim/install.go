// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/install"
)

func newInstallCmd(logger logrus.FieldLogger) *cobra.Command {
	return &cobra.Command{
		Use:  "install",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return install.Install(logger)
		},
	}
}
