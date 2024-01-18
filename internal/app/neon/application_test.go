// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	"log"
	"testing"
)

func TestApplicationCheck(t *testing.T) {
	type fields struct {
		config  *config
		logger  *log.Logger
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
			name: "default",
			fields: fields{
				config: &config{
					Store: &configStore{
						Config: map[string]interface{}{
							"storage": map[string]map[string]interface{}{
								"test": {},
							},
						},
					},
					Fetcher: &configFetcher{},
					Loader:  &configLoader{},
					Server: &configServer{
						Config: map[string]interface{}{
							"listeners": map[string]map[string]interface{}{
								"test": {},
							},
							"sites": map[string]map[string]interface{}{
								"default": {
									"listeners": []string{"test"},
									"routes":    map[string]interface{}{},
								},
							},
						},
					},
				},
				logger:  log.Default(),
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
