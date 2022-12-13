// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package static

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kubelet/pkg/apis/credentialprovider/install"
	"k8s.io/kubelet/pkg/apis/credentialprovider/v1beta1"

	"github.com/mesosphere/dynamic-credential-provider/pkg/credentialprovider/plugin"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

//nolint:gochecknoinits // init is idiomatically used to set up schemes
func init() {
	install.Install(scheme)
}

// staticProvider implements the credential provider interface, reading a credentials file on the disk.
type staticProvider struct {
	credentialsFile string
}

// NewProvider creates a new instance of the static credentials provider.
func NewProvider(credentialsFile string) (plugin.CredentialProvider, error) {
	return staticProvider{credentialsFile: credentialsFile}, nil
}

// GetCredentials will ignore the image and args arguments and simply read a credentials file and return its content.
func (s staticProvider) GetCredentials(
	_ context.Context,
	_ string,
	_ []string,
) (response *v1beta1.CredentialProviderResponse, err error) {
	credentials, err := os.ReadFile(s.credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("error reading credentials file: %w", err)
	}

	return DecodeResponse(credentials)
}

func DecodeResponse(data []byte) (*v1beta1.CredentialProviderResponse, error) {
	obj, gvk, err := codecs.UniversalDecoder(v1beta1.SchemeGroupVersion).Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	if gvk.Kind != "CredentialProviderResponse" {
		return nil, fmt.Errorf("kind was %q, expected CredentialProviderResponse", gvk.Kind)
	}

	if gvk.Group != v1beta1.GroupName {
		return nil, fmt.Errorf("group was %q, expected %s", gvk.Group, v1beta1.GroupName)
	}

	response, ok := obj.(*v1beta1.CredentialProviderResponse)
	if !ok {
		return nil, fmt.Errorf("unable to convert %T to *CredentialProviderResponse", obj)
	}

	return response, nil
}
