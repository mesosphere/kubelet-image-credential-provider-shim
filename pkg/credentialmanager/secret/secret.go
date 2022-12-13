// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/mesosphere/dynamic-credential-provider/pkg/credentialmanager/plugin"
	credentialproviderplugin "github.com/mesosphere/dynamic-credential-provider/pkg/credentialprovider/plugin"
	"github.com/mesosphere/dynamic-credential-provider/pkg/credentialprovider/static"
)

//nolint:gosec // No credentials here.
const (
	SecretName      = "staticcredentialproviderauth"
	SecretNamespace = "kube-system"

	SecretKeyName = "static-image-credentials.json"
)

type CredentialManager struct {
	client kubernetes.Interface

	name      string
	namespace string
	key       string
}

func NewSecretsCredentialManager(client kubernetes.Interface) plugin.CredentialManager {
	return &CredentialManager{
		client:    client,
		name:      SecretName,
		namespace: SecretNamespace,
		key:       SecretKeyName,
	}
}

func (m *CredentialManager) Update(
	ctx context.Context,
	address, username, password string,
) error {
	secrets := m.client.CoreV1().Secrets(
		m.namespace,
	)
	secret, err := secrets.Get(
		ctx,
		m.name,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("unable to get secret: %w", err)
	}

	err = m.updateSecret(secret, address, username, password)
	if err != nil {
		return err
	}

	_, err = secrets.Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("unable to update secret: %w", err)
	}

	return nil
}

func (m *CredentialManager) updateSecret(
	secret *v1.Secret,
	address, username, password string,
) error {
	data, found := secret.Data[m.key]
	if !found {
		return fmt.Errorf("secret %s/%s exists, but missing key %q", m.namespace, m.name, m.key)
	}

	credentialProviderResponse, err := static.DecodeResponse(data)
	if err != nil {
		return fmt.Errorf("failed to decode Secret data: %w", err)
	}

	auth, found := credentialProviderResponse.Auth[address]
	if !found {
		return fmt.Errorf(
			"secret %s/%s exists, but missing entry for registry %q",
			m.namespace,
			m.name,
			address,
		)
	}
	auth.Username = username
	auth.Password = password

	credentialProviderResponse.Auth[address] = auth

	data, err = credentialproviderplugin.EncodeResponse(credentialProviderResponse)
	if err != nil {
		return fmt.Errorf("failed to encode Secret data: %w", err)
	}

	// Secret found. Update it.
	secret.Data[m.key] = data

	return nil
}
