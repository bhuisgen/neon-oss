// Copyright 2022-2023 Boris HUISGEN. All rights reserved.
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
					Listeners: []*configListener{
						{
							Name: "listener",
							Config: map[string]interface{}{
								"test": map[string]interface{}{},
							},
						},
					},
					Servers: []*configServer{
						{
							Name: "server",
							Config: map[string]interface{}{
								"listeners": []string{"default"},
							},
						},
					},
					Store:   &configStore{},
					Fetcher: &configFetcher{},
					Loader:  &configLoader{},
				},
				logger:  log.Default(),
				store:   &store{},
				fetcher: &fetcher{},
				loader:  &loader{},
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
			}
			if err := a.Check(); (err != nil) != tt.wantErr {
				t.Errorf("application.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
