// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package plugin

import "context"

type CredentialManager interface {
	Update(ctx context.Context, address, username, password string) error
}
