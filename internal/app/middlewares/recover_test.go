// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package middlewares

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRecover(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://test", nil)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	recover := Recover(&RecoverConfig{Writer: io.Discard}, next)
	recover.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestRecoverPanic(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://test", nil)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})
	recover := Recover(&RecoverConfig{Writer: io.Discard}, next)
	recover.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}
}

func TestRecoverStack(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://test", nil)

	os.Setenv("DEBUG", "1")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})
	recover := Recover(&RecoverConfig{Writer: io.Discard}, next)
	recover.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}
}
