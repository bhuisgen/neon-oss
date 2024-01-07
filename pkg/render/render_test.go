// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package render

import (
	"net/http"
	"reflect"
	"testing"
)

func TestRenderBody(t *testing.T) {
	type fields struct {
		body        []byte
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "default",
			fields: fields{
				body: []byte("test"),
			},
			want: []byte("test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &render{
				body:        tt.fields.body,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			if got := r.Body(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("render.Body() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderHeader(t *testing.T) {
	type fields struct {
		body        []byte
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
		want   http.Header
	}{
		{
			name: "default",
			fields: fields{
				header: http.Header{
					"test": []string{"test"},
				},
			},
			want: http.Header{
				"test": []string{"test"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &render{
				body:        tt.fields.body,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			if got := r.Header(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("render.Header() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderStatusCode(t *testing.T) {
	type fields struct {
		body        []byte
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "default",
			fields: fields{
				statusCode: http.StatusOK,
			},
			want: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &render{
				body:        tt.fields.body,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			if got := r.StatusCode(); got != tt.want {
				t.Errorf("render.StatusCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderRedirect(t *testing.T) {
	type fields struct {
		body        []byte
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			fields: fields{
				redirect: true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &render{
				body:        tt.fields.body,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			if got := r.Redirect(); got != tt.want {
				t.Errorf("render.Redirect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderRedirectURL(t *testing.T) {
	type fields struct {
		body        []byte
		header      http.Header
		statusCode  int
		redirect    bool
		redirectURL string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "default",
			fields: fields{
				redirectURL: "/redirect",
			},
			want: "/redirect",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &render{
				body:        tt.fields.body,
				header:      tt.fields.header,
				statusCode:  tt.fields.statusCode,
				redirect:    tt.fields.redirect,
				redirectURL: tt.fields.redirectURL,
			}
			if got := r.RedirectURL(); got != tt.want {
				t.Errorf("render.RedirectURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
