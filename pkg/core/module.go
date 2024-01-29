// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

import (
	"log/slog"

	"github.com/bhuisgen/neon/pkg/module"
)

// Module
type Module interface {
	module.Module
	Init(config map[string]interface{}, logger *slog.Logger) error
}
