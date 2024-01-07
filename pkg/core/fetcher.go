// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

import "context"

// Fetcher
type Fetcher interface {
	Fetch(ctx context.Context, name string, provider string, config map[string]interface{}) (*Resource, error)
}

// FetcherProviderModule
type FetcherProviderModule interface {
	Module
	Fetch(ctx context.Context, name string, config map[string]interface{}) (*Resource, error)
}
