
package sessions

import "net/http"

// Options stores configuration for a sessions or sessions store.
//
// Fields are a subset of http.Cookie fields.
type Options struct {
	Path   string
	Domain string
	// MaxAge=0 means no Max-Age attribute specified and the cookie will be
	// deleted after the browser sessions ends.
	// MaxAge<0 means delete cookie immediately.
	// MaxAge>0 means Max-Age attribute present and given in seconds.
	MaxAge   int
	Secure   bool
	HttpOnly bool
	// Defaults to http.SameSiteDefaultMode
	SameSite http.SameSite
}
