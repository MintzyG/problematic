package fp

import "encoding/json"

// standardMembers are the RFC 9457 base members. An extension can never
// replace one of them.
var standardMembers = map[string]bool{
	"type":     true,
	"status":   true,
	"title":    true,
	"detail":   true,
	"instance": true,
}

// MarshalJSON implements json.Marshaler.
//
// Standard fields are marshaled normally via the Alias trick to avoid
// infinite recursion. Extension members are then merged into the same
// top-level JSON object, producing a flat application/problem+json document
// as required by RFC 9457 §3. An extension whose key is a standard member
// is skipped, so the struct fields always win.
func (p *Problem) MarshalJSON() ([]byte, error) {
	type Alias Problem
	b, err := json.Marshal((*Alias)(p))
	if err != nil {
		return nil, err
	}
	if len(p.Extensions) == 0 {
		return b, nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	for k, v := range p.Extensions {
		if !standardMembers[k] {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// UnmarshalJSON implements json.Unmarshaler.
//
// Standard fields are decoded into the struct normally. Any additional
// top-level keys not belonging to the RFC 9457 base schema are collected
// into [Problem.Extensions] as raw JSON, preserving unknown extension
// members without data loss.
func (p *Problem) UnmarshalJSON(data []byte) error {
	type Alias Problem
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for k, v := range m {
		if !standardMembers[k] {
			if p.Extensions == nil {
				p.Extensions = make(map[string]json.RawMessage)
			}
			p.Extensions[k] = v
		}
	}
	return nil
}
