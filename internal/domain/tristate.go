package domain

import (
	"encoding/json"
	"fmt"
)

// Tristate models a boolean fact that may be unobservable.
//
// The zero value is Unknown on purpose: a struct that was never populated
// must never claim that a security feature is disabled.
type Tristate uint8

const (
	Unknown Tristate = iota
	False
	True
)

func Bool(v bool) Tristate {
	if v {
		return True
	}
	return False
}

func (t Tristate) Known() bool { return t != Unknown }

func (t Tristate) String() string {
	switch t {
	case True:
		return "true"
	case False:
		return "false"
	default:
		return "unknown"
	}
}

func (t Tristate) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Tristate) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "true":
		*t = True
	case "false":
		*t = False
	case "unknown":
		*t = Unknown
	default:
		return fmt.Errorf("domain: invalid tristate %q", s)
	}
	return nil
}

// OptInt is an integer fact that may be unobservable.
type OptInt struct {
	Value int
	Valid bool
}

func SomeInt(v int) OptInt { return OptInt{Value: v, Valid: true} }

func (o OptInt) MarshalJSON() ([]byte, error) {
	if !o.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(o.Value)
}

func (o OptInt) String() string {
	if !o.Valid {
		return "unknown"
	}
	return fmt.Sprintf("%d", o.Value)
}

func (o *OptInt) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*o = OptInt{}
		return nil
	}
	var value int
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	*o = SomeInt(value)
	return nil
}
