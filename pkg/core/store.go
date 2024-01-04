// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

// Store
type Store interface {
	Get(name string) (*Resource, error)
	Set(name string, resource *Resource) error
}
