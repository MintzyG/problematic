// Package main demonstrates basic construction and JSON serialization of
// RFC 9457 Problem Details using the fp package.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	fp "github.com/MintzyG/fun/problems"
)

func main() {
	// --- 1. Simple problem ---------------------------------------------------
	p := fp.New(404).
		WithType("https://example.com/errors/not-found").
		WithDetail("user 42 does not exist").
		WithInstance("/users/42")

	encode(p)

	// --- 2. Problem with extension members -----------------------------------
	type FieldError struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	}

	validation := fp.New(422).
		WithType("https://example.com/errors/validation").
		WithTitle("Validation Failed").
		WithExtension("errors", []FieldError{
			{Field: "email", Message: "required"},
			{Field: "name", Message: "must be at least 2 characters"},
		})

	encode(validation)

	// --- 3. Sentinel + Clone pattern -----------------------------------------
	var ErrForbidden = fp.New(403).
		WithType("https://example.com/errors/forbidden")

	// Clone so the sentinel is never mutated.
	p2 := ErrForbidden.Clone().WithDetail("you do not have access to this resource")
	encode(p2)

	// Sentinel is unchanged.
	_, _ = fmt.Fprintln(os.Stderr, "sentinel Detail:", ErrForbidden.Detail) // ""
}

func encode(p *fp.Problem) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "encode error:", err)
	}
}
