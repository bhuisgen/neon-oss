package render

import "net/http"

// Render is the interface of a render.
type Render interface {
	// Body returns the HTTP response body.
	Body() []byte
	// Header returns the HTTP response headers.
	Header() http.Header
	// StatusCode returns the HTTP response status code.
	StatusCode() int
	// Redirect returns the redirect flag.
	Redirect() bool
	// RedirectURL returns the redirect URL.
	RedirectURL() string
}

// render implements a render.
type render struct {
	body        []byte
	header      http.Header
	statusCode  int
	redirect    bool
	redirectURL string
}

// Body returns the HTTP response body.
func (r *render) Body() []byte {
	return r.body
}

// Header returns the HTTP response headers.
func (r *render) Header() http.Header {
	return r.header
}

// StatusCode returns the HTTP response status code.
func (r *render) StatusCode() int {
	return r.statusCode
}

// Redirect returns the redirect flag.
func (r *render) Redirect() bool {
	return r.redirect
}

// RedirectURL returns the redirect URL.
func (r *render) RedirectURL() string {
	return r.redirectURL
}

var _ Render = (*render)(nil)
