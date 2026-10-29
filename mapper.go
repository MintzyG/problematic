package fp

import (
	"errors"
	"net/http"
)

// ErrorMapper is a function that attempts to convert an arbitrary error into
// a [Problem]. It should return nil if it does not recognize the error, so
// that [FromError] can try the next mapper in the chain.
type ErrorMapper func(err error) *Problem

// FromError converts any error into a [Problem] using a chain of mappers.
//
// Resolution order:
//  1. If err already is (or wraps) a *Problem via errors.As, that Problem is
//     returned directly.
//  2. Each mapper is tried in order; the first non-nil result wins.
//  3. If no mapper matches, a generic 500 Internal Server Error Problem is
//     returned. Its Detail is the status text, never err's message, so an
//     unmapped error cannot leak internals to the client; log err yourself.
//
// This makes FromError safe to call unconditionally in HTTP handlers:
//
//	p := fp.FromError(err, myDomainMapper)
//	_ = fp.Write(w, p)
func FromError(err error, mappers ...ErrorMapper) *Problem {
	if p, ok := errors.AsType[*Problem](err); ok {
		return p
	}
	for _, m := range mappers {
		if p := m(err); p != nil {
			return p
		}
	}
	return New(http.StatusInternalServerError).
		WithDetail(http.StatusText(http.StatusInternalServerError))
}
