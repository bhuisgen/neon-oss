package js

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bhuisgen/gomonkey"
)

// apiSite adds the site API.
func (v *vm) apiSite(ctx *gomonkey.Context, server *gomonkey.Object) error {
	site, err := ctx.DefineObject(server, "site", 0)
	if err != nil {
		return err
	}
	defer site.Release()

	name := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		return gomonkey.NewValueString(ctx, v.config.Site.Name())
	}
	if err := ctx.DefineFunction(site, "name", name, 0, 0); err != nil {
		return err
	}

	listeners := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		items := make([]*gomonkey.Value, 0, len(v.config.Site.Listeners()))
		defer func() {
			for _, item := range items {
				item.Release()
			}
		}()
		for _, listener := range v.config.Site.Listeners() {
			item, err := gomonkey.NewValueString(ctx, listener)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		arr, err := gomonkey.NewArrayObject(ctx, items...)
		if err != nil {
			return nil, err
		}
		return arr.AsValue(), nil
	}
	if err := ctx.DefineFunction(site, "listeners", listeners, 0, 0); err != nil {
		return err
	}

	hosts := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		items := make([]*gomonkey.Value, 0, len(v.config.Site.Hosts()))
		defer func() {
			for _, item := range items {
				item.Release()
			}
		}()
		for _, host := range v.config.Site.Hosts() {
			item, err := gomonkey.NewValueString(ctx, host)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		arr, err := gomonkey.NewArrayObject(ctx, items...)
		if err != nil {
			return nil, err
		}
		return arr.AsValue(), nil
	}
	if err := ctx.DefineFunction(site, "hosts", hosts, 0, 0); err != nil {
		return err
	}

	isDefault := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		return gomonkey.NewValueBoolean(ctx, v.config.Site.IsDefault())
	}
	if err := ctx.DefineFunction(site, "isDefault", isDefault, 0, 0); err != nil {
		return err
	}

	return nil
}

// apiHandler add the handler API.
func (v *vm) apiHandler(ctx *gomonkey.Context, server *gomonkey.Object) error {
	handler, err := ctx.DefineObject(server, "handler", 0)
	if err != nil {
		return err
	}
	defer handler.Release()

	state := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		o, err := gomonkey.JSONParse(ctx, string(v.config.State))
		if err != nil {
			return nil, err
		}
		return o.AsValue(), nil
	}
	if err := ctx.DefineFunction(handler, "state", state, 0, 0); err != nil {
		return err
	}

	return nil
}

// apiRequest adds the request API.
func (v *vm) apiRequest(ctx *gomonkey.Context, server *gomonkey.Object) error {
	request, err := ctx.DefineObject(server, "request", 0)
	if err != nil {
		return err
	}
	defer request.Release()

	method := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		return gomonkey.NewValueString(ctx, v.config.Request.Method)
	}
	if err := ctx.DefineFunction(request, "method", method, 0, 0); err != nil {
		return err
	}

	proto := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		return gomonkey.NewValueString(ctx, v.config.Request.Proto)
	}
	if err := ctx.DefineFunction(request, "proto", proto, 0, 0); err != nil {
		return err
	}

	protoMajor := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		return gomonkey.NewValueInt32(ctx, int32(v.config.Request.ProtoMajor))
	}
	if err := ctx.DefineFunction(request, "protoMajor", protoMajor, 0, 0); err != nil {
		return err
	}

	protoMinor := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		return gomonkey.NewValueInt32(ctx, int32(v.config.Request.ProtoMinor))
	}
	if err := ctx.DefineFunction(request, "protoMinor", protoMinor, 0, 0); err != nil {
		return err
	}

	remoteAddr := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		return gomonkey.NewValueString(ctx, v.config.Request.RemoteAddr)
	}
	if err := ctx.DefineFunction(request, "remoteAddr", remoteAddr, 0, 0); err != nil {
		return err
	}

	host := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		return gomonkey.NewValueString(ctx, v.config.Request.Host)
	}
	if err := ctx.DefineFunction(request, "host", host, 0, 0); err != nil {
		return err
	}

	path := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		return gomonkey.NewValueString(ctx, v.config.Request.URL.Path)
	}
	if err := ctx.DefineFunction(request, "path", path, 0, 0); err != nil {
		return err
	}

	query := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		query := v.config.Request.URL.Query()
		data, err := json.Marshal(&query)
		if err != nil {
			return nil, err
		}
		return gomonkey.NewValueString(ctx, string(data))
	}
	if err := ctx.DefineFunction(request, "query", query, 0, 0); err != nil {
		return err
	}

	headers := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		headers := v.config.Request.Header
		data, err := json.Marshal(&headers)
		if err != nil {
			return nil, err
		}
		return gomonkey.NewValueString(ctx, string(data))
	}
	if err := ctx.DefineFunction(request, "headers", headers, 0, 0); err != nil {
		return err
	}

	return nil
}

// apiResponse adds the response API.
func (v *vm) apiResponse(ctx *gomonkey.Context, server *gomonkey.Object) error {
	response, err := ctx.DefineObject(server, "response", 0)
	if err != nil {
		return err
	}
	defer response.Release()

	render := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		if len(args) < 1 {
			return nil, errors.New("invalid arguments")
		}
		if !args[0].IsString() {
			return nil, errors.New("invalid content")
		}
		content := []byte(args[0].ToString())
		status := http.StatusOK
		if len(args) >= 2 && args[1].IsInt32() {
			code := args[1].ToInt32()
			if code < 100 || code > 599 {
				return nil, errors.New("invalid code")
			}
			status = int(code)
		}

		v.data.render = content
		v.data.status = status

		return nil, nil
	}
	if err := ctx.DefineFunction(response, "render", render, 0, 0); err != nil {
		return err
	}

	redirect := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		if len(args) < 1 {
			return nil, errors.New("invalid arguments")
		}
		if !args[0].IsString() {
			return nil, errors.New("invalid url")
		}
		redirect := true
		redirectURL := args[0].ToString()
		redirectStatus := http.StatusFound
		if len(args) >= 2 && args[1].IsInt32() {
			code := args[1].ToInt32()
			if code < 100 || code > 599 {
				return nil, errors.New("invalid status")
			}
			redirectStatus = int(code)
		}

		v.data.redirect = redirect
		v.data.redirectURL = redirectURL
		v.data.redirectStatus = redirectStatus

		return nil, nil
	}
	if err := ctx.DefineFunction(response, "redirect", redirect, 0, 0); err != nil {
		return err
	}

	setHeader := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		if len(args) < 2 {
			return nil, errors.New("invalid arguments")
		}
		if !args[0].IsString() {
			return nil, errors.New("invalid key")
		}
		if !args[1].IsString() {
			return nil, errors.New("invalid value")
		}
		key := args[0].ToString()
		value := args[1].ToString()

		if v.data.headers == nil {
			v.data.headers = make(map[string][]string)
		}
		v.data.headers[key] = append(v.data.headers[key], value)

		return nil, nil
	}
	if err := ctx.DefineFunction(response, "setHeader", setHeader, 0, 0); err != nil {
		return err
	}

	setTitle := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		if len(args) < 1 {
			return nil, errors.New("invalid arguments")
		}
		if !args[0].IsString() {
			return nil, errors.New("invalid title")
		}
		title := args[0].ToString()

		v.data.title = &title

		return nil, nil
	}
	if err := ctx.DefineFunction(response, "setTitle", setTitle, 0, 0); err != nil {
		return err
	}

	setMeta := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		if len(args) < 2 {
			return nil, errors.New("invalid arguments")
		}
		if !args[0].IsString() {
			return nil, errors.New("invalid id")
		}
		if !args[1].IsObject() {
			return nil, errors.New("invalid attributes")
		}
		id := args[0].ToString()
		attributes, err := args[1].AsObject()
		if err != nil {
			return nil, nil
		}

		e := newDOMElement(id)
		entries, err := attributes.Call("entries")
		if err != nil {
			return nil, nil
		}
		defer entries.Release()
		iterator, err := entries.AsObject()
		if err != nil {
			return nil, nil
		}
		for {
			next, err := iterator.Call("next")
			if err != nil {
				return nil, nil
			}
			defer next.Release()
			iteration, err := next.AsObject()
			if err != nil {
				return nil, nil
			}
			done, err := iteration.Get("done")
			if err != nil {
				return nil, nil
			}
			defer done.Release()
			if done.ToBoolean() {
				break
			}
			value, err := iteration.Get("value")
			if err != nil {
				return nil, nil
			}
			defer value.Release()
			array, err := value.AsObject()
			if err != nil {
				return nil, nil
			}
			k, err := array.GetElement(0)
			if err != nil {
				return nil, nil
			}
			defer k.Release()
			v, err := array.GetElement(1)
			if err != nil {
				return nil, nil
			}
			defer v.Release()
			e.SetAttribute(k.String(), v.String())
		}
		if v.data.metas == nil {
			v.data.metas = newDOMElementList()
		}
		v.data.metas.Set(e)

		return nil, nil
	}
	if err := ctx.DefineFunction(response, "setMeta", setMeta, 0, 0); err != nil {
		return err
	}

	setLink := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		if len(args) < 2 {
			return nil, errors.New("invalid arguments")
		}
		if !args[0].IsString() {
			return nil, errors.New("invalid id")
		}
		if !args[1].IsObject() {
			return nil, errors.New("invalid attributes")
		}
		id := args[0].ToString()
		attributes, err := args[1].AsObject()
		if err != nil {
			return nil, nil
		}

		e := newDOMElement(id)
		entries, err := attributes.Call("entries")
		if err != nil {
			return nil, nil
		}
		defer entries.Release()
		iterator, err := entries.AsObject()
		if err != nil {
			return nil, nil
		}
		for {
			next, err := iterator.Call("next")
			if err != nil {
				return nil, nil
			}
			defer next.Release()
			iteration, err := next.AsObject()
			if err != nil {
				return nil, nil
			}
			done, err := iteration.Get("done")
			if err != nil {
				return nil, nil
			}
			defer done.Release()
			if done.ToBoolean() {
				break
			}
			value, err := iteration.Get("value")
			if err != nil {
				return nil, nil
			}
			defer value.Release()
			array, err := value.AsObject()
			if err != nil {
				return nil, nil
			}
			k, err := array.GetElement(0)
			if err != nil {
				return nil, nil
			}
			defer k.Release()
			v, err := array.GetElement(1)
			if err != nil {
				return nil, nil
			}
			defer v.Release()
			e.SetAttribute(k.String(), v.String())
		}
		if v.data.links == nil {
			v.data.links = newDOMElementList()
		}
		v.data.links.Set(e)

		return nil, nil
	}
	if err := ctx.DefineFunction(response, "setLink", setLink, 0, 0); err != nil {
		return err
	}

	setScript := func(args []*gomonkey.Value) (*gomonkey.Value, error) {
		if len(args) < 2 {
			return nil, errors.New("invalid arguments")
		}
		if !args[0].IsString() {
			return nil, errors.New("invalid id")
		}
		if !args[1].IsObject() {
			return nil, errors.New("invalid attributes")
		}
		id := args[0].ToString()
		attributes, err := args[1].AsObject()
		if err != nil {
			return nil, nil
		}

		e := newDOMElement(id)
		entries, err := attributes.Call("entries")
		if err != nil {
			return nil, nil
		}
		defer entries.Release()
		iterator, err := entries.AsObject()
		if err != nil {
			return nil, nil
		}
		for {
			next, err := iterator.Call("next")
			if err != nil {
				return nil, nil
			}
			defer next.Release()
			iteration, err := next.AsObject()
			if err != nil {
				return nil, nil
			}
			done, err := iteration.Get("done")
			if err != nil {
				return nil, nil
			}
			defer done.Release()
			if done.ToBoolean() {
				break
			}
			value, err := iteration.Get("value")
			if err != nil {
				return nil, nil
			}
			defer value.Release()
			array, err := value.AsObject()
			if err != nil {
				return nil, nil
			}
			k, err := array.GetElement(0)
			if err != nil {
				return nil, nil
			}
			defer k.Release()
			v, err := array.GetElement(1)
			if err != nil {
				return nil, nil
			}
			defer v.Release()
			e.SetAttribute(k.String(), v.String())
		}
		if v.data.scripts == nil {
			v.data.scripts = newDOMElementList()
		}
		v.data.scripts.Set(e)

		return nil, nil
	}
	if err := ctx.DefineFunction(response, "setScript", setScript, 0, 0); err != nil {
		return err
	}

	return nil
}
