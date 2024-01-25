// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"encoding/json"
	"net/http"
	"strconv"

	"rogchap.com/v8go"
)

// api injects all JS APIs to the given VM.
func api(v *vm) error {
	if err := apiHandler(v); err != nil {
		return err
	}
	if err := apiRequest(v); err != nil {
		return err
	}
	if err := apiResponse(v); err != nil {
		return err
	}
	return nil
}

// apiHandler injects the handler API.
func apiHandler(v *vm) error {
	var err error
	err = v.serverHandlerObject.Set("state", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, err := v8go.JSONParse(v.context, *v.config.State)
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	return nil
}

// apiRequest injects the request API.
func apiRequest(v *vm) error {
	var err error

	err = v.serverRequestObject.Set("method", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, err := v8go.NewValue(v.isolate, v.config.Request.Method)
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverRequestObject.Set("proto", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, err := v8go.NewValue(v.isolate, v.config.Request.Proto)
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverRequestObject.Set("protoMajor", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, err := v8go.NewValue(v.isolate, v.config.Request.ProtoMajor)
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverRequestObject.Set("protoMinor", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, err := v8go.NewValue(v.isolate, v.config.Request.ProtoMinor)
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverRequestObject.Set("remoteAddr", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, err := v8go.NewValue(v.isolate, v.config.Request.RemoteAddr)
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverRequestObject.Set("host", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, err := v8go.NewValue(v.isolate, v.config.Request.Host)
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverRequestObject.Set("path", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			value, err := v8go.NewValue(v.isolate, v.config.Request.URL.Path)
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverRequestObject.Set("query", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			var value *v8go.Value
			q := v.config.Request.URL.Query()
			data, err := json.Marshal(&q)
			if err != nil {
				return nil
			}
			value, err = v8go.JSONParse(v.context, string(data))
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverRequestObject.Set("headers", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			var value *v8go.Value
			h := v.config.Request.Header
			data, err := json.Marshal(&h)
			if err != nil {
				return nil
			}
			value, _ = v8go.JSONParse(v.context, string(data))
			if err != nil {
				return nil
			}
			return value
		}))
	if err != nil {
		return vmErrConfigure
	}

	return nil
}

// apiResponse injects the response API.
func apiResponse(v *vm) error {
	var err error

	err = v.serverResponseObject.Set("render", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 1 || !args[0].IsString() {
				return nil
			}

			r := []byte(args[0].String())
			v.data.render = &r

			status := http.StatusOK
			if len(args) > 1 && args[1].IsNumber() {
				code, err := strconv.Atoi(args[1].String())
				if err != nil || code < 100 || code > 599 {
					code = http.StatusInternalServerError
				}
				status = code
			}
			v.data.status = &status

			return nil
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverResponseObject.Set("redirect", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 1 || !args[0].IsString() {
				return nil
			}

			redirect := true
			redirectURL := args[0].String()
			v.data.redirect = &redirect
			v.data.redirectURL = &redirectURL

			redirectStatus := http.StatusFound
			if len(args) > 1 && args[1].IsNumber() {
				code, err := strconv.Atoi(args[1].String())
				if err != nil || code < 100 || code > 599 {
					code = http.StatusInternalServerError
				}
				redirectStatus = code
			}
			v.data.redirectStatus = &redirectStatus

			return nil
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverResponseObject.Set("setHeader", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 2 || !args[0].IsString() || !args[1].IsString() {
				return nil
			}

			key := args[0].String()
			value := args[1].String()

			if v.data.headers == nil {
				v.data.headers = make(map[string][]string)
			}
			v.data.headers[key] = append(v.data.headers[key], value)

			return nil
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverResponseObject.Set("setTitle", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 1 || !args[0].IsString() {
				return nil
			}

			title := args[0].String()
			v.data.title = &title

			return nil
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverResponseObject.Set("setMeta", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 2 || !args[0].IsString() || !args[1].IsMap() {
				return nil
			}

			id := args[0].String()
			attributes, err := args[1].AsObject()
			if err != nil {
				return nil
			}

			e := newDOMElement(id)

			entries, err := attributes.MethodCall("entries")
			if err != nil {
				return nil
			}
			iterator, err := entries.AsObject()
			if err != nil {
				return nil
			}
			for {
				next, err := iterator.MethodCall("next")
				if err != nil {
					return nil
				}
				iteration, err := next.AsObject()
				if err != nil {
					return nil
				}
				done, err := iteration.Get("done")
				if err != nil {
					return nil
				}
				if done.Boolean() {
					break
				}
				value, err := iteration.Get("value")
				if err != nil {
					return nil
				}
				array, err := value.AsObject()
				if err != nil {
					return nil
				}
				k, err := array.GetIdx(0)
				if err != nil || !k.IsString() {
					return nil
				}
				v, err := array.GetIdx(1)
				if err != nil || !v.IsString() {
					return nil
				}
				e.SetAttribute(k.String(), v.String())
			}

			if v.data.metas == nil {
				v.data.metas = newDOMElementList()
			}
			v.data.metas.Set(e)

			return nil
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverResponseObject.Set("setLink", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 2 || !args[0].IsString() || !args[1].IsMap() {
				return nil
			}

			id := args[0].String()
			attributes, err := args[1].AsObject()
			if err != nil {
				return nil
			}

			e := newDOMElement(id)

			entries, err := attributes.MethodCall("entries")
			if err != nil {
				return nil
			}
			iterator, err := entries.AsObject()
			if err != nil {
				return nil
			}
			for {
				next, err := iterator.MethodCall("next")
				if err != nil {
					return nil
				}
				iteration, err := next.AsObject()
				if err != nil {
					return nil
				}
				done, err := iteration.Get("done")
				if err != nil {
					return nil
				}
				if done.Boolean() {
					break
				}
				value, err := iteration.Get("value")
				if err != nil {
					return nil
				}
				array, err := value.AsObject()
				if err != nil {
					return nil
				}
				k, err := array.GetIdx(0)
				if err != nil || !k.IsString() {
					return nil
				}
				v, err := array.GetIdx(1)
				if err != nil || !v.IsString() {
					return nil
				}
				e.SetAttribute(k.String(), v.String())
			}

			if v.data.links == nil {
				v.data.links = newDOMElementList()
			}
			v.data.links.Set(e)

			return nil
		}))
	if err != nil {
		return vmErrConfigure
	}

	err = v.serverResponseObject.Set("setScript", v.v8NewFunctionTemplate(v.isolate,
		func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 2 || !args[0].IsString() || !args[1].IsMap() {
				return nil
			}

			id := args[0].String()
			attributes, err := args[1].AsObject()
			if err != nil {
				return nil
			}

			e := newDOMElement(id)

			entries, err := attributes.MethodCall("entries")
			if err != nil {
				return nil
			}
			iterator, err := entries.AsObject()
			if err != nil {
				return nil
			}
			for {
				next, err := iterator.MethodCall("next")
				if err != nil {
					return nil
				}
				iteration, err := next.AsObject()
				if err != nil {
					return nil
				}
				done, err := iteration.Get("done")
				if err != nil {
					return nil
				}
				if done.Boolean() {
					break
				}
				value, err := iteration.Get("value")
				if err != nil {
					return nil
				}
				array, err := value.AsObject()
				if err != nil {
					return nil
				}
				k, err := array.GetIdx(0)
				if err != nil || !k.IsString() {
					return nil
				}
				v, err := array.GetIdx(1)
				if err != nil || !v.IsString() {
					return nil
				}
				e.SetAttribute(k.String(), v.String())
			}

			if v.data.scripts == nil {
				v.data.scripts = newDOMElementList()
			}
			v.data.scripts.Set(e)

			return nil
		}))
	if err != nil {
		return vmErrConfigure
	}

	return nil
}
