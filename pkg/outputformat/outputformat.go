package outputformat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	toon "github.com/toon-format/toon-go"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

const (
	FormatJSON = "json"
	FormatTOON = "toon"
)

// Normalize canonicalizes structured output format values and defaults to JSON.
// Note: commands that have a human-readable default (listen, logs tail, whoami)
// should check for empty-string before calling Normalize/Validate — empty means
// "use the default non-structured output", not "use JSON".
func Normalize(format string) string {
	if strings.TrimSpace(format) == "" {
		return FormatJSON
	}

	return strings.ToLower(strings.TrimSpace(format))
}

// Validate ensures the format is one of the supported structured output values.
func Validate(format string) error {
	switch Normalize(format) {
	case FormatJSON, FormatTOON:
		return nil
	default:
		return fmt.Errorf("invalid format %q, must be one of 'json' or 'toon'", format)
	}
}

// RequestFlagUsage returns the help text for response-format flags whose
// default behavior is JSON output.
func RequestFlagUsage() string {
	return "Specifies the response format\n" +
		"Acceptable values:\n" +
		"  'json' - Output the response in JSON format (default)\n" +
		"  'toon' - Output the response in TOON format"
}

// StructuredFlagUsage returns the help text for structured format flags.
func StructuredFlagUsage(subject string) string {
	return fmt.Sprintf("Specifies the output format of %s\nAcceptable values:\n  'json' - Output %s in JSON format\n  'toon' - Output %s in TOON format", subject, subject, subject)
}

// Marshal renders a Go value as either indented JSON or TOON. The TOON path
// goes through JSON first so existing json tags define the emitted shape.
func Marshal(v any, format string) ([]byte, error) {
	if err := Validate(format); err != nil {
		return nil, err
	}

	switch Normalize(format) {
	case FormatTOON:
		jsonData, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}

		ordered, err := decodeOrderedJSON(jsonData)
		if err != nil {
			return nil, err
		}

		return toon.Marshal(ordered)
	default:
		return json.MarshalIndent(v, "", "  ")
	}
}

// RenderJSON formats a raw JSON payload as either colorized JSON or TOON.
func RenderJSON(raw []byte, format string, darkStyle bool, w io.Writer) (string, error) {
	if err := Validate(format); err != nil {
		return "", err
	}

	switch Normalize(format) {
	case FormatTOON:
		ordered, err := decodeOrderedJSON(raw)
		if err != nil {
			return "", err
		}

		data, err := toon.Marshal(ordered)
		if err != nil {
			return "", err
		}

		return string(data), nil
	default:
		return ansi.ColorizeJSON(string(raw), darkStyle, w), nil
	}
}

func decodeOrderedJSON(raw []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	value, err := decodeOrderedValue(dec)
	if err != nil {
		return nil, err
	}

	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("invalid JSON: unexpected trailing data")
		}
		return nil, err
	}

	return value, nil
}

func decodeOrderedValue(dec *json.Decoder) (any, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}

	switch tok := token.(type) {
	case json.Delim:
		switch tok {
		case '{':
			return decodeOrderedObject(dec)
		case '[':
			return decodeOrderedArray(dec)
		default:
			return nil, fmt.Errorf("invalid JSON: unexpected delimiter %q", tok)
		}
	default:
		return tok, nil
	}
}

func decodeOrderedObject(dec *json.Decoder) (toon.Object, error) {
	fields := make([]toon.Field, 0)

	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return toon.Object{}, err
		}

		key, ok := keyToken.(string)
		if !ok {
			return toon.Object{}, fmt.Errorf("invalid JSON object key %T", keyToken)
		}

		value, err := decodeOrderedValue(dec)
		if err != nil {
			return toon.Object{}, err
		}

		fields = append(fields, toon.Field{
			Key:   key,
			Value: value,
		})
	}

	if err := expectDelimiter(dec, '}'); err != nil {
		return toon.Object{}, err
	}

	return toon.NewObject(fields...), nil
}

func decodeOrderedArray(dec *json.Decoder) ([]any, error) {
	values := make([]any, 0)

	for dec.More() {
		value, err := decodeOrderedValue(dec)
		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	if err := expectDelimiter(dec, ']'); err != nil {
		return nil, err
	}

	return values, nil
}

func expectDelimiter(dec *json.Decoder, want json.Delim) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}

	got, ok := token.(json.Delim)
	if !ok || got != want {
		return fmt.Errorf("invalid JSON: expected delimiter %q, got %v", want, token)
	}

	return nil
}
