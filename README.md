# fp — `fun/problems`

RFC 9457 Problem Details for HTTP APIs in Go.

## Overview

`fp` is a small, zero-dependency library that implements the [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) `application/problem+json` format. It gives you a structured, serializable error type that carries an HTTP status code, a machine-readable type URI, human-readable title and detail strings, an optional instance URI, and arbitrary extension members, all in one flat JSON object.

```json
{
  "type": "https://example.com/errors/validation",
  "status": 422,
  "title": "Validation Failed",
  "detail": "email is required",
  "instance": "/users",
  "errors": [
    { "field": "email", "message": "required" }
  ]
}
```

## Installation

```sh
go get github.com/MintzyG/fun/problems
```

Import with a short alias:

```go
import fp "github.com/MintzyG/fun/problems"
```

## Usage

### Constructing a Problem

```go
p := fp.New(http.StatusNotFound).
    WithType("https://example.com/errors/not-found").
    WithDetail("user 42 does not exist").
    WithInstance("/users/42")
```

`New` sets `type` to `"about:blank"` and `title` to the canonical HTTP status text. All `With*` methods mutate and return the receiver for chaining.

### Extension members

RFC 9457 allows arbitrary additional members at the top level of the problem object. Use `WithExtension` to add them — they are marshalled alongside the standard fields, not nested:

```go
p := fp.New(422).
    WithType("https://example.com/errors/validation").
    WithExtension("errors", []FieldError{
        {Field: "email", Message: "required"},
    })
```

### Problems as errors

`*Problem` implements the `error` interface. `Error()` returns `"Title: Detail"`. You can wrap and unwrap Problems with the standard `errors` package:

```go
if err := doSomething(); err != nil {
    return fmt.Errorf("handler: %w", fp.New(500).WithDetail(err.Error()))
}

// later
var p *fp.Problem
if errors.As(err, &p) {
    log.Printf("HTTP %d: %s", p.Status, p.Detail)
}
```

### Sentinel Problems + Clone

Define package-level sentinels and clone them for each occurrence to avoid mutation:

```go
var ErrNotFound = fp.New(404).
    WithType("https://example.com/errors/not-found")

func getUser(id int) error {
    // ...
    return ErrNotFound.Clone().WithDetail(fmt.Sprintf("user %d does not exist", id))
}
```

### Mapping errors in HTTP handlers

`FromError` converts any `error` into a `*Problem` using a chain of `ErrorMapper` functions:

```go
func domainMapper(err error) *fp.Problem {
    switch {
    case errors.Is(err, ErrNotFound):
        return fp.New(404).WithDetail(err.Error())
    case errors.Is(err, ErrForbidden):
        return fp.New(403).WithDetail(err.Error())
    }
    return nil // not recognised — try the next mapper
}

func handler(w http.ResponseWriter, r *http.Request) {
    user, err := repo.GetUser(r.Context(), id)
    if err != nil {
        p := fp.FromError(err, domainMapper)
        w.Header().Set("Content-Type", "application/problem+json")
        w.WriteHeader(p.Status)
        json.NewEncoder(w).Encode(p)
        return
    }
    // ...
}
```

Resolution order in `FromError`:

1. If `err` is (or wraps) a `*Problem`, return it directly.
2. Try each `ErrorMapper` in order; first non-nil result wins.
3. Fall back to a 500 Problem with `err.Error()` as the detail.

## JSON behaviour

- **Marshal**: standard fields + extension members are flattened into a single top-level object.
- **Unmarshal**: standard fields are decoded into the struct; any unknown top-level keys are collected into `Extensions` as `json.RawMessage`, so extension data is never silently dropped.

## Examples

- [`examples/basic`](examples/basic/main.go) — construction, chaining, Clone pattern.
- [`examples/http_handler`](examples/http_handler/main.go) — `FromError` + `ErrorMapper` in a `net/http` handler.

## RFC 9457 field reference

| Field      | Type   | Description                                                    |
|------------|--------|----------------------------------------------------------------|
| `type`     | string | URI identifying the problem type. Defaults to `"about:blank"`. |
| `status`   | int    | HTTP status code.                                              |
| `title`    | string | Short, stable summary of the problem type.                     |
| `detail`   | string | Occurrence-specific human-readable explanation.                |
| `instance` | string | URI identifying this specific occurrence. Optional.            |
| *(any)*    | any    | Extension members — arbitrary additional fields.               |

## License

Part of the [fun](https://github.com/MintzyG/fun) library suite.