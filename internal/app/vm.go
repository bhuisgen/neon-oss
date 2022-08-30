// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"net/http"
	"strconv"

	"rogchap.com/v8go"
)

// vm implements a VM
type vm struct {
	isolate                  *v8go.Isolate
	processObject            *v8go.ObjectTemplate
	envObject                *v8go.ObjectTemplate
	serverObject             *v8go.ObjectTemplate
	serverDataObject         *v8go.ObjectTemplate
	serverDataResourceObject *v8go.ObjectTemplate
	title                    *string
	metas                    map[string]map[string]string
	links                    map[string]map[string]string
	scripts                  map[string]map[string]string
	redirect                 *bool
	redirectURL              *string
	redirectStatus           *int
	render                   *[]byte
	status                   *int
}

// NewVM creates a new VM
func NewVM() *vm {
	isolate := v8go.NewIsolate()

	processObject := v8go.NewObjectTemplate(isolate)
	envObject := v8go.NewObjectTemplate(isolate)
	serverObject := v8go.NewObjectTemplate(isolate)
	serverDataObject := v8go.NewObjectTemplate(isolate)
	serverDataResourceObject := v8go.NewObjectTemplate(isolate)

	var title string
	serverObject.Set("setTitle", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			title = args[0].String()
		}
		return nil
	}))

	var metas map[string]map[string]string = make(map[string]map[string]string)
	serverObject.Set("setMeta", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 1 {
			id := args[0].String()
			data := args[1].Object()

			metas[id] = make(map[string]string)
			for _, attribute := range []string{"name", "itemprop", "content", "charset", "http-equiv", "scheme", "property"} {
				if ok := data.Has(attribute); ok {
					v, _ := data.Get(attribute)
					metas[id][attribute] = v.String()
				}
			}
		}
		return nil
	}))

	var links map[string]map[string]string = make(map[string]map[string]string)
	serverObject.Set("setLink", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 1 {
			id := args[0].String()
			data := args[1].Object()

			links[id] = make(map[string]string)
			for _, attribute := range []string{"rel", "href", "hreflang", "type", "sizes", "media", "as",
				"crossorigin", "disabled", "importance", "integrity", "referrerpolicy", "title"} {
				if ok := data.Has(attribute); ok {
					v, _ := data.Get(attribute)
					links[id][attribute] = v.String()
				}
			}
		}
		return nil
	}))

	var scripts map[string]map[string]string = make(map[string]map[string]string)
	serverObject.Set("setScript", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 1 {
			id := args[0].String()
			data := args[1].Object()

			scripts[id] = make(map[string]string)
			for _, attribute := range []string{"type", "src", "async", "crossorigin", "defer", "integrity", "nomodule",
				"nonce", "referrerpolicy", "children"} {
				if ok := data.Has(attribute); ok {
					v, _ := data.Get(attribute)
					scripts[id][attribute] = v.String()
				}
			}
		}
		return nil
	}))

	var redirect bool
	var redirectURL string
	var redirectStatus int = http.StatusTemporaryRedirect
	serverObject.Set("redirect", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			redirect = true
			redirectURL = args[0].String()
			if len(args) > 1 {
				redirectStatus, _ = strconv.Atoi(args[1].String())
			}
		}
		return nil
	}))

	var render []byte
	var status int = http.StatusOK
	serverObject.Set("render", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			render = []byte(args[0].String())
			if len(args) > 1 {
				status, _ = strconv.Atoi(args[1].String())
			}
		}
		return nil
	}))

	serverDataResourceObject.Set("loading", false)
	serverDataResourceObject.Set("error", v8go.Null(isolate))
	serverDataResourceObject.Set("response", "{}")

	return &vm{
		isolate:                  isolate,
		processObject:            processObject,
		envObject:                envObject,
		serverObject:             serverObject,
		serverDataObject:         serverDataObject,
		serverDataResourceObject: serverDataResourceObject,
		title:                    &title,
		metas:                    metas,
		links:                    links,
		scripts:                  scripts,
		redirect:                 &redirect,
		redirectURL:              &redirectURL,
		redirectStatus:           &redirectStatus,
		render:                   &render,
		status:                   &status,
	}
}

// Close closes the VM
func (v *vm) Close() {
	v.isolate.Dispose()
}

// Configure configures the VM
func (v *vm) Configure() {
	*v.title = ""
	*v.redirect = false
	*v.redirectURL = ""
	*v.redirectStatus = 0
	*v.render = []byte{}
}
