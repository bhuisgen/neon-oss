// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

import "context"

// LoaderParserModule
type LoaderParserModule interface {
	Module
	Parse(ctx context.Context, store Store, fetcher Fetcher) error
}
