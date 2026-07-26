package flux

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// marshalYAML renders v as a single YAML document with a stable,
// human-reviewable 2-space indent. Byte-identical output across runs on
// the same input is a tested contract (golden rules 22, 45).
func marshalYAML(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
