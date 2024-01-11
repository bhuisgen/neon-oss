// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

// Cache
type Cache interface {
	Get(key string) any
	Set(key string, value any)
	Remove(key string)
	Clear()
}
