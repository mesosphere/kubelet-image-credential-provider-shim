// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	//nolint:lll // Just a long string.
	initialResponse = []byte(
		`{"kind":"CredentialProviderResponse","apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1","cacheKeyType":"Image","cacheDuration":"0s","auth":{"docker.io":{"username":"initialusername","password":"initialpassword"}}}
`,
	)
	//nolint:lll // Just a long string.
	updatedResponse = []byte(
		`{"kind":"CredentialProviderResponse","apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1","cacheKeyType":"Image","cacheDuration":"0s","auth":{"docker.io":{"username":"newusername","password":"newpassword"}}}
`,
	)
)

func Test_updateSecret(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		address      string
		username     string
		password     string
		initialData  map[string][]byte
		expectedData map[string][]byte
		expectErr    error
	}{
		{
			name:         "credentials are updated",
			address:      "docker.io",
			username:     "newusername",
			password:     "newpassword",
			initialData:  map[string][]byte{SecretKeyName: initialResponse},
			expectedData: map[string][]byte{SecretKeyName: updatedResponse},
		},
		{
			name:         "credentials not updated, missing Secret key",
			address:      "docker.io",
			username:     "newusername",
			password:     "newpassword",
			initialData:  map[string][]byte{"wrong-key": initialResponse},
			expectedData: map[string][]byte{"wrong-key": initialResponse},
			expectErr: fmt.Errorf(
				"secret kube-system/staticcredentialproviderauth exists, but missing key \"static-image-credentials.json\"",
			),
		},
		{
			name:         "credentials not updated, missing registry entry",
			address:      "docker.com",
			username:     "newusername",
			password:     "newpassword",
			initialData:  map[string][]byte{SecretKeyName: initialResponse},
			expectedData: map[string][]byte{SecretKeyName: initialResponse},
			expectErr: fmt.Errorf(
				"secret kube-system/staticcredentialproviderauth exists, but missing entry for registry \"docker.com\"",
			),
		},
	}

	for _, tt := range testcases {
		tt := tt // Capture range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manager := testManager()
			secret := testSecret(tt.initialData)
			err := manager.updateSecret(secret, tt.address, tt.username, tt.password)
			assert.Equal(t, tt.expectErr, err)

			assert.Equal(
				t,
				string(tt.expectedData[SecretKeyName]),
				string(secret.Data[SecretKeyName]),
			)
		})
	}
}

func testManager() *CredentialManager {
	return &CredentialManager{
		name:      SecretName,
		namespace: SecretNamespace,
		key:       SecretKeyName,
	}
}

func testSecret(data map[string][]byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretName,
			Namespace: SecretNamespace,
		},
		Data: data,
	}
}
