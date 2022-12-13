// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/mesosphere/dkp-cli-runtime/core/cmd/version"
)

type k8sClientCreateError struct {
	err error
}

func (e k8sClientCreateError) Error() string {
	return fmt.Sprintf("unable to create kubernetes client: %v", e.err)
}

func NewFromKubeconfig(kubeconfig string) (kubernetes.Interface, clientcmd.ClientConfig, error) {
	config, clientConfig, err := clientConfigsFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, k8sClientCreateError{err: err}
	}
	return clientset, clientConfig, nil
}

func clientConfigsFromKubeconfig(kubeconfig string) (*rest.Config, clientcmd.ClientConfig, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(
		loadingRules,
		overrides,
		nil,
	)

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, k8sClientCreateError{err: err}
	}

	config.UserAgent = UserAgent()
	return config, clientConfig, nil
}

func UserAgent() string {
	return fmt.Sprintf("credential-manager/%s", version.GetVersion().GitVersion)
}
