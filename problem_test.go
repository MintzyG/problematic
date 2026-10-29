package fp_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	fp "github.com/MintzyG/problematic"
)

// ── New ──────────────────────────────────────────────────────────────────────

func TestNew_SetsDefaults(t *testing.T) {
	p := fp.New(http.StatusNotFound)
	if p.Status != 404 {
		t.Errorf("Status: want 404, got %d", p.Status)
	}
	if p.Title != "Not Found" {
		t.Errorf("Title: want %q, got %q", "Not Found", p.Title)
	}
	if p.Type != "about:blank" {
		t.Errorf("Type: want %q, got %q", "about:blank", p.Type)
	}
}

func TestNew_CoercesInvalidStatus(t *testing.T) {
	for _, s := range []int{0, 200, 399, 600, 999} {
		p := fp.New(s)
		if p.Status != 500 {
			t.Errorf("New(%d).Status: want 500, got %d", s, p.Status)
		}
	}
}

func TestNew_AcceptsBoundaryStatuses(t *testing.T) {
	for _, s := range []int{400, 499, 500, 599} {
		p := fp.New(s)
		if p.Status != s {
			t.Errorf("New(%d).Status: want %d, got %d", s, s, p.Status)
		}
	}
}

// ── Builder chain ─────────────────────────────────────────────────────────────

func TestWith_Chain(t *testing.T) {
	p := fp.New(422).
		WithType("https://example.com/errors/validation").
		WithTitle("Validation Failed").
		WithDetail("email is required").
		WithInstance("/users")

	if p.Type != "https://example.com/errors/validation" {
		t.Errorf("Type mismatch: %q", p.Type)
	}
	if p.Title != "Validation Failed" {
		t.Errorf("Title mismatch: %q", p.Title)
	}
	if p.Detail != "email is required" {
		t.Errorf("Detail mismatch: %q", p.Detail)
	}
	if p.Instance != "/users" {
		t.Errorf("Instance mismatch: %q", p.Instance)
	}
}

// ── error interface ───────────────────────────────────────────────────────────

func TestError_Format(t *testing.T) {
	p := fp.New(404).WithDetail("not found")
	want := "Not Found: not found"
	if p.Error() != want {
		t.Errorf("Error(): want %q, got %q", want, p.Error())
	}
}

func TestError_ErrorsAs(t *testing.T) {
	p := fp.New(403).WithDetail("forbidden")
	wrapped := errors.Join(errors.New("outer"), p)

	var target *fp.Problem
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should unwrap to *Problem")
	}
	if target.Status != 403 {
		t.Errorf("unwrapped Status: want 403, got %d", target.Status)
	}
}

// ── Clone ─────────────────────────────────────────────────────────────────────

func TestClone_IsDeepCopy(t *testing.T) {
	orig := fp.New(404).
		WithDetail("original").
		WithExtension("foo", "bar")

	clone := orig.Clone()
	clone.Detail = "mutated"
	clone.Extensions["foo"] = json.RawMessage(`"baz"`)

	if orig.Detail != "original" {
		t.Error("Clone mutated original Detail")
	}
	if string(orig.Extensions["foo"]) != `"bar"` {
		t.Error("Clone mutated original Extensions")
	}
}

func TestClone_NilExtensions(t *testing.T) {
	orig := fp.New(500)
	clone := orig.Clone()
	_ = clone.WithExtension("x", 1)
	if orig.Extensions != nil {
		t.Error("Clone should not share Extensions when original had none")
	}
}

// ── JSON round-trip ───────────────────────────────────────────────────────────

func TestMarshalJSON_BasicFields(t *testing.T) {
	p := fp.New(404).WithDetail("not found")
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("Unmarshal into map: %v", err)
	}
	if _, ok := m["status"]; !ok {
		t.Error("missing 'status' key")
	}
	if _, ok := m["title"]; !ok {
		t.Error("missing 'title' key")
	}
	if _, ok := m["detail"]; !ok {
		t.Error("missing 'detail' key")
	}
}

func TestMarshalJSON_ExtensionsFlattened(t *testing.T) {
	p := fp.New(422).WithExtension("errors", []string{"a", "b"})
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("Unmarshal into map: %v", err)
	}
	if _, ok := m["errors"]; !ok {
		t.Error("extension 'errors' not flattened into top-level object")
	}
}

func TestUnmarshalJSON_RoundTrip(t *testing.T) {
	orig := fp.New(409).
		WithType("https://example.com/conflict").
		WithDetail("duplicate key").
		WithInstance("/items/1").
		WithExtension("retryAfter", 30)

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got fp.Problem
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Status != orig.Status {
		t.Errorf("Status: want %d, got %d", orig.Status, got.Status)
	}
	if got.Type != orig.Type {
		t.Errorf("Type: want %q, got %q", orig.Type, got.Type)
	}
	if got.Detail != orig.Detail {
		t.Errorf("Detail: want %q, got %q", orig.Detail, got.Detail)
	}
	if got.Instance != orig.Instance {
		t.Errorf("Instance: want %q, got %q", orig.Instance, got.Instance)
	}
	if _, ok := got.Extensions["retryAfter"]; !ok {
		t.Error("extension 'retryAfter' missing after round-trip")
	}
}

func TestUnmarshalJSON_UnknownExtensionsCollected(t *testing.T) {
	raw := `{"type":"about:blank","status":400,"title":"Bad Request","detail":"oops","custom":"value","count":3}`
	var p fp.Problem
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := p.Extensions["custom"]; !ok {
		t.Error("'custom' should be in Extensions")
	}
	if _, ok := p.Extensions["count"]; !ok {
		t.Error("'count' should be in Extensions")
	}
	// standard fields must NOT appear as extensions
	if _, ok := p.Extensions["status"]; ok {
		t.Error("'status' should not be in Extensions")
	}
}

// ── WithExtension ─────────────────────────────────────────────────────────────

func TestWithExtension_InitialisesMap(t *testing.T) {
	p := fp.New(400)
	if p.Extensions != nil {
		t.Fatal("Extensions should be nil before first WithExtension call")
	}
	_ = p.WithExtension("key", "val")
	if p.Extensions == nil {
		t.Error("Extensions should be initialised after WithExtension")
	}
}

func TestWithExtension_OverwritesKey(t *testing.T) {
	p := fp.New(400).WithExtension("k", 1).WithExtension("k", 2)
	var v int
	if err := json.Unmarshal(p.Extensions["k"], &v); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if v != 2 {
		t.Errorf("want 2, got %d", v)
	}
}

// ── FromError ─────────────────────────────────────────────────────────────────

func TestFromError_ReturnsProblemDirectly(t *testing.T) {
	orig := fp.New(404).WithDetail("not found")
	got := fp.FromError(orig)
	if got != orig {
		t.Error("FromError should return the same *Problem when err is a *Problem")
	}
}

func TestFromError_UnwrapsWrappedProblem(t *testing.T) {
	orig := fp.New(403)
	wrapped := errors.Join(errors.New("context"), orig)
	got := fp.FromError(wrapped)
	if got != orig {
		t.Error("FromError should unwrap *Problem from a wrapped error")
	}
}

func TestFromError_TriesMappers(t *testing.T) {
	sentinel := errors.New("sentinel")
	mapper := func(err error) *fp.Problem {
		if errors.Is(err, sentinel) {
			return fp.New(404).WithDetail("mapped")
		}
		return nil
	}

	got := fp.FromError(sentinel, mapper)
	if got.Status != 404 {
		t.Errorf("Status: want 404, got %d", got.Status)
	}
	if got.Detail != "mapped" {
		t.Errorf("Detail: want %q, got %q", "mapped", got.Detail)
	}
}

func TestFromError_FallsBackTo500(t *testing.T) {
	got := fp.FromError(errors.New("something exploded"))
	if got.Status != 500 {
		t.Errorf("Status: want 500, got %d", got.Status)
	}
}

func TestFromError_FallbackHidesErrorMessage(t *testing.T) {
	got := fp.FromError(errors.New("pq: password authentication failed"))
	if got.Detail != "Internal Server Error" {
		t.Errorf("Detail: want %q, got %q", "Internal Server Error", got.Detail)
	}
}

// ── extensions vs standard members ───────────────────────────────────────────

func TestMarshalJSON_ExtensionCannotShadowStandardMember(t *testing.T) {
	p := fp.New(404).WithDetail("real").
		WithExtension("status", 200).
		WithExtension("detail", "fake").
		WithExtension("extra", "kept")

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["status"] != float64(404) {
		t.Errorf("status: want 404, got %v", m["status"])
	}
	if m["detail"] != "real" {
		t.Errorf("detail: want %q, got %v", "real", m["detail"])
	}
	if m["extra"] != "kept" {
		t.Errorf("extra: want %q, got %v", "kept", m["extra"])
	}
}

func TestMarshalJSON_AlwaysWritesStandardMembers(t *testing.T) {
	b, err := json.Marshal(fp.New(404))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"type", "status", "title", "detail", "instance"} {
		if _, ok := m[k]; !ok {
			t.Errorf("member %q missing from %s", k, b)
		}
	}
}

func TestWithExtension_UnmarshalableValueIsNull(t *testing.T) {
	p := fp.New(400).WithExtension("bad", make(chan int))
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if v, ok := m["bad"]; !ok || v != nil {
		t.Errorf("bad: want null, got %v (present %v)", v, ok)
	}
}

// ── Write ────────────────────────────────────────────────────────────────────

func TestWrite_SetsHeaderStatusAndBody(t *testing.T) {
	rec := httptest.NewRecorder()
	p := fp.New(http.StatusConflict).WithDetail("slug taken").WithInstance("/projects")

	if err := fp.Write(rec, p); err != nil {
		t.Fatal(err)
	}
	if got := rec.Header().Get("Content-Type"); got != fp.ContentType {
		t.Errorf("Content-Type: want %q, got %q", fp.ContentType, got)
	}
	if rec.Code != http.StatusConflict {
		t.Errorf("status: want 409, got %d", rec.Code)
	}
	var got fp.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != 409 || got.Detail != "slug taken" || got.Instance != "/projects" {
		t.Errorf("body: got %+v", got)
	}
}

func TestFromError_MapperChainStopsAtFirstMatch(t *testing.T) {
	calls := 0
	mk := func(status int) fp.ErrorMapper {
		return func(err error) *fp.Problem {
			calls++
			return fp.New(status)
		}
	}
	_ = fp.FromError(errors.New("e"), mk(400), mk(401))
	if calls != 1 {
		t.Errorf("expected 1 mapper call, got %d", calls)
	}
}
