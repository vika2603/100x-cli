package output

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// JSON writes the payload as JSON to stdout, optionally filtered through
// the renderer's gojq expression.
func (r *Renderer) JSON(payload any) error {
	if r.JQ == "" {
		enc := json.NewEncoder(r.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	q, err := gojq.Parse(r.JQ)
	if err != nil {
		return fmt.Errorf("parse --jq: %w", err)
	}
	// Round-trip through encoding/json so gojq sees a generic Go value.
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	iter := q.Run(v)
	enc := json.NewEncoder(r.Out)
	enc.SetIndent("", "  ")
	for {
		got, ok := iter.Next()
		if !ok {
			return nil
		}
		if err, ok := got.(error); ok {
			return err
		}
		if err := enc.Encode(got); err != nil {
			return err
		}
	}
}
