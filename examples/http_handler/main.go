// Package main demonstrates integrating fp with a net/http handler using
// FromError and custom ErrorMappers to translate domain errors into
// RFC 9457 Problem Details responses.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	fp "github.com/MintzyG/fun/problems"
)

// ── domain errors ─────────────────────────────────────────────────────────────

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

// domainMapper translates known domain errors into Problems.
// Return nil for unrecognized errors so FromError tries the next mapper.
func domainMapper(err error) *fp.Problem {
	switch {
	case errors.Is(err, ErrNotFound):
		return fp.New(http.StatusNotFound).
			WithType("https://example.com/errors/not-found").
			WithDetail(err.Error())
	case errors.Is(err, ErrForbidden):
		return fp.New(http.StatusForbidden).
			WithType("https://example.com/errors/forbidden").
			WithDetail(err.Error())
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// writeProblem writes a Problem as application/problem+json.
func writeProblem(w http.ResponseWriter, p *fp.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// ── handlers ──────────────────────────────────────────────────────────────────

func getUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		p := fp.New(http.StatusBadRequest).
			WithType("https://example.com/errors/missing-param").
			WithDetail("query parameter 'id' is required").
			WithInstance(r.URL.Path)
		writeProblem(w, p)
		return
	}
	if id == "99" {
		// Simulate a domain error bubbling up.
		writeProblem(w, fp.FromError(ErrForbidden, domainMapper))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"id":%q,"name":"Alice"}`, id)
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", getUser)

	fmt.Println("listening on :8080")
	fmt.Println("  GET /users?id=1   → 200 JSON")
	fmt.Println("  GET /users        → 400 Problem")
	fmt.Println("  GET /users?id=99  → 403 Problem")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
