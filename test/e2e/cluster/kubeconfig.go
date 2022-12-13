// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"fmt"
	"os"
)

func KubeconfigFromString(clusterName, kubeconfig string) (string, error) {
	kubeconfigFile, err := os.CreateTemp(
		"",
		fmt.Sprintf("%s-kindcluster-kubeconfig-*", clusterName),
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to create temporary file for KinD cluster kubeconfig: %w",
			err,
		)
	}

	//nolint:revive // 0400 is standard read-only perms.
	return kubeconfigFile.Name(), os.WriteFile(kubeconfigFile.Name(), []byte(kubeconfig), 0o400)
}
