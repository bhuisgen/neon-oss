// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package core

// Store
type Store interface {
	LoadResource(name string) (*Resource, error)
	StoreResource(name string, resource *Resource) error
}

// StoreStorageModule
type StoreStorageModule interface {
	Module
	LoadResource(name string) (*Resource, error)
	StoreResource(name string, resource *Resource) error
}
