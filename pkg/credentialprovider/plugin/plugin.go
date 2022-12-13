// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/kubelet/pkg/apis/credentialprovider/v1beta1"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

//nolint:gochecknoinits // init is idiomatically used to set up schemes
func init() {
	utilruntime.Must(v1beta1.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1beta1.SchemeGroupVersion))
}

// CredentialProvider is an interface implemented by the kubelet credential provider plugin to fetch
// the username/password based on the provided image name.
type CredentialProvider interface {
	GetCredentials(
		ctx context.Context,
		image string,
		args []string,
	) (*v1beta1.CredentialProviderResponse, error)
}

// ExecPlugin implements the exec-based plugin for fetching credentials that is invoked by the kubelet.
type ExecPlugin struct {
	plugin CredentialProvider
}

// NewProvider returns an instance of execPlugin that fetches
// credentials based on the provided plugin implementing the CredentialProvider interface.
func NewProvider(plugin CredentialProvider) *ExecPlugin {
	return &ExecPlugin{plugin}
}

// Run executes the credential provider plugin. Required information for the plugin request (in
// the form of v1beta1.CredentialProviderRequest) is provided via stdin from the kubelet.
// The CredentialProviderResponse, containing the username/password required for pulling
// the provided image, will be sent back to the kubelet via stdout.
func (e *ExecPlugin) Run(ctx context.Context) error {
	return e.runPlugin(ctx, os.Stdin, os.Stdout, os.Args[1:])
}

var (
	ErrEmptyImageInRequest           = errors.New("image in plugin request was empty")
	ErrNilCredentialProviderResponse = errors.New("CredentialProviderResponse from plugin was nil")
	ErrUnsupportedAPIVersion         = errors.New("unsupported API version")
)

func (e *ExecPlugin) runPlugin(ctx context.Context, r io.Reader, w io.Writer, args []string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	gvk, err := json.DefaultMetaFactory.Interpret(data)
	if err != nil {
		return err
	}

	if gvk.GroupVersion() != v1beta1.SchemeGroupVersion {
		return fmt.Errorf("%w: %s", ErrUnsupportedAPIVersion, gvk)
	}

	request, err := decodeRequest(data)
	if err != nil {
		return err
	}

	if request.Image == "" {
		return fmt.Errorf("%w", ErrEmptyImageInRequest)
	}

	response, err := e.plugin.GetCredentials(ctx, request.Image, args)
	if err != nil {
		return err
	}

	if response == nil {
		return fmt.Errorf("%w", ErrNilCredentialProviderResponse)
	}

	encodedResponse, err := EncodeResponse(response)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(w)
	defer writer.Flush()
	if _, err := writer.Write(encodedResponse); err != nil {
		return err
	}

	return nil
}

var (
	ErrUnsupportedRequestKind = errors.New(
		"unsupported credential provider request kind",
	)
	ErrConversionFailure = errors.New("conversion failure")
)

func decodeRequest(data []byte) (*v1beta1.CredentialProviderRequest, error) {
	obj, gvk, err := codecs.UniversalDecoder(v1beta1.SchemeGroupVersion).Decode(data, nil, nil)
	if err != nil {
		if runtime.IsNotRegisteredError(err) {
			return nil, fmt.Errorf("%w: %v", ErrUnsupportedRequestKind, err)
		}
		return nil, err
	}

	if gvk.Kind != "CredentialProviderRequest" {
		return nil, fmt.Errorf(
			"%w: %s (expected CredentialProviderRequest)",
			ErrUnsupportedRequestKind,
			gvk.Kind,
		)
	}

	if gvk.Group != v1beta1.GroupName {
		return nil, fmt.Errorf(
			"%w: %s (expected %s)",
			ErrUnsupportedAPIVersion, gvk.GroupVersion(), v1beta1.SchemeGroupVersion,
		)
	}

	request, ok := obj.(*v1beta1.CredentialProviderRequest)
	if !ok {
		return nil, fmt.Errorf(
			"%w: unable to convert %T to *CredentialProviderRequest",
			ErrConversionFailure,
			obj,
		)
	}

	return request, nil
}

func EncodeResponse(response *v1beta1.CredentialProviderResponse) ([]byte, error) {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return nil, fmt.Errorf("unsupported media type %q", mediaType)
	}

	encoder := codecs.EncoderForVersion(info.Serializer, v1beta1.SchemeGroupVersion)
	data, err := runtime.Encode(encoder, response)
	if err != nil {
		return nil, fmt.Errorf("failed to encode response: %v", err)
	}

	return data, nil
}
