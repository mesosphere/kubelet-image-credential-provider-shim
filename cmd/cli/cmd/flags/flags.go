// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package flags

import (
	"io"

	"github.com/mesosphere/dkp-cli-runtime/core/output"
)

// CLIConfig injects dependencies into CLI that are hard to mock,
// enabling better unittesting.
type CLIConfig struct {
	In     io.Reader
	Output output.Output
}
