// Package fp implements RFC 9457 "Problem Details for HTTP APIs".
//
// A [Problem] is a structured error type that carries an HTTP status code,
// a machine-readable type URI, a human-readable title and detail, an
// optional instance URI, and arbitrary extension members, all serialized
// as application/problem+json.
//
// Basic usage:
//
//	p := fp.New(http.StatusNotFound).
//	    WithType("https://example.com/errors/not-found").
//	    WithDetail("user 42 does not exist").
//	    WithInstance("/users/42")
//
// Problems implement the error interface, so they can be returned from
// any function that returns error and unwrapped with errors.As.
package fp

import (
	"encoding/json"
	"net/http"
)

// Problem is an RFC 9457 Problem Details object.
//
// The zero value is not meaningful; use [New] to construct one.
//
// All With* methods mutate the receiver and return it so calls can be chained:
//
//	return fp.New(422).WithDetail("email is required")
//
// Problems implement the error interface; Error returns "Title: Detail".
type Problem struct {
	// Type is a URI reference that identifies the problem type.
	// Defaults to "about:blank" when constructed via [New], which means
	// the Title field alone describes the problem class.
	Type string `json:"type"`

	// Status mirrors the HTTP status code used for the response.
	Status int `json:"status"`

	// Title is a short, human-readable summary of the problem type.
	// It SHOULD NOT change between occurrences of the same problem type.
	// [New] sets this to the standard text for the given status code.
	Title string `json:"title"`

	// Detail is a human-readable explanation specific to this occurrence
	// of the problem. It MAY differ between occurrences of the same type.
	Detail string `json:"detail"`

	// Instance is a URI reference that identifies the specific occurrence
	// of the problem. Optional.
	Instance string `json:"instance"`

	// Extensions holds any additional members present in the
	// application/problem+json object that are not part of the base schema.
	// Keys must not collide with the standard fields above.
	// Use [Problem.WithExtension] to add entries.
	Extensions map[string]json.RawMessage `json:"-"`
}

// Error implements the error interface. It returns "Title: Detail".
func (p *Problem) Error() string {
	return p.Title + ": " + p.Detail
}

// New constructs a Problem with the given HTTP status code.
// The Type is set to "about:blank" and Title is set to the canonical
// status text (e.g. "Not Found" for 404).
//
// If status is outside the 4xx–5xx range it is coerced to 500.
func New(status int) *Problem {
	if status < 400 || status >= 600 {
		status = 500
	}
	return &Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
	}
}

// WithType sets the Type URI for this Problem and returns the receiver.
// Per RFC 9457 §3.1.1 the value SHOULD be an absolute URI; consumers
// dereference it to obtain human-readable documentation for the problem type.
func (p *Problem) WithType(t string) *Problem {
	p.Type = t
	return p
}

// WithInstance sets the Instance URI for this Problem and returns the receiver.
// Per RFC 9457 §3.1.5 the value identifies the specific occurrence of the
// problem, for example the request path that triggered it.
func (p *Problem) WithInstance(instance string) *Problem {
	p.Instance = instance
	return p
}

// WithTitle overrides the default Title for this Problem and returns the receiver.
// Use sparingly — the Title is intended to be stable across all occurrences of
// a given Type; prefer [WithDetail] for occurrence-specific information.
func (p *Problem) WithTitle(title string) *Problem {
	p.Title = title
	return p
}

// WithDetail sets the human-readable detail string for this Problem and returns
// the receiver. Detail MAY differ between occurrences of the same Type and
// SHOULD focus on what went wrong in this specific case.
func (p *Problem) WithDetail(detail string) *Problem {
	p.Detail = detail
	return p
}

// WithExtension adds an RFC 9457 extension member to the Problem and returns
// the receiver. value is JSON-marshaled; panicking on marshal failure is
// intentional — extension values should always be serializable.
//
// Keys must not shadow the standard fields ("type", "status", "title",
// "detail", "instance").
//
//	p.WithExtension("errors", validationErrors)
func (p *Problem) WithExtension(key string, value any) *Problem {
	if p.Extensions == nil {
		p.Extensions = make(map[string]json.RawMessage)
	}
	b, _ := json.Marshal(value)
	p.Extensions[key] = b
	return p
}

// Clone returns a deep copy of the Problem. Extensions are copied so that
// mutations to the clone do not affect the original — useful when using a
// package-level sentinel Problem as a template.
//
//	var ErrNotFound = fp.New(404).WithType("https://example.com/errors/not-found")
//
//	// safe — won't mutate the sentinel
//	return ErrNotFound.Clone().WithDetail("user 42 does not exist")
func (p *Problem) Clone() *Problem {
	np := *p
	if p.Extensions != nil {
		np.Extensions = make(map[string]json.RawMessage, len(p.Extensions))
		for k, v := range p.Extensions {
			np.Extensions[k] = v
		}
	}
	return &np
}
