// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"log/slog"
	"testing"
)

func TestApplicationCheck(t *testing.T) {
	type fields struct {
		config  *config
		logger  *slog.Logger
		state   *applicationState
		store   *store
		fetcher *fetcher
		loader  *loader
		server  *server
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "minimal",
			fields: fields{
				config: &config{
					Store: &configStore{
						Config: nil,
					},
					Fetcher: &configFetcher{
						Config: nil,
					},
					Loader: &configLoader{
						Config: nil,
					},
					Server: &configServer{
						Config: map[string]interface{}{
							"listeners": map[string]interface{}{
								"default": map[string]interface{}{
									"test": map[string]interface{}{
										"option": "value",
									},
								},
							},
							"sites": map[string]interface{}{
								"main": map[string]interface{}{
									"listeners": []string{"default"},
									"routes":    map[string]interface{}{},
								},
							},
						},
					},
				},
				logger:  slog.Default(),
				store:   &store{},
				fetcher: &fetcher{},
				loader:  &loader{},
				server:  &server{},
			},
		},
		{
			name: "default",
			fields: fields{
				config: &config{
					Store: &configStore{
						Config: map[string]interface{}{
							"storage": map[string]interface{}{
								"test": map[string]interface{}{},
							},
						},
					},
					Fetcher: &configFetcher{
						Config: map[string]interface{}{
							"providers": map[string]interface{}{
								"test": map[string]interface{}{},
							},
						},
					},
					Loader: &configLoader{
						Config: map[string]interface{}{},
					},
					Server: &configServer{
						Config: map[string]interface{}{
							"listeners": map[string]interface{}{
								"default": map[string]interface{}{
									"test": map[string]interface{}{
										"option": "value",
									},
								},
							},
							"sites": map[string]interface{}{
								"main": map[string]interface{}{
									"listeners": []string{"default"},
									"routes":    map[string]interface{}{},
								},
							},
						},
					},
				},
				logger:  slog.Default(),
				store:   &store{},
				fetcher: &fetcher{},
				loader:  &loader{},
				server:  &server{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &application{
				config:  tt.fields.config,
				logger:  tt.fields.logger,
				state:   tt.fields.state,
				store:   tt.fields.store,
				fetcher: tt.fields.fetcher,
				loader:  tt.fields.loader,
				server:  tt.fields.server,
			}
			if err := a.Check(); (err != nil) != tt.wantErr {
				t.Errorf("application.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
