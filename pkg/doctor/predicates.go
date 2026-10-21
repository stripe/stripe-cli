package doctor

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
)

const maxInput = 1 << 20

//go:embed predicates.json
var mockPredicates []byte

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,79}$`)

type Declaration struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Version int             `json:"version"`
	Options json.RawMessage `json:"options"`
}

type LegacyFix struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Param string `json:"param"`
}

type Condition struct {
	Operator string      `json:"operator"`
	Key      string      `json:"key,omitempty"`
	Value    string      `json:"value,omitempty"`
	Items    []Condition `json:"items,omitempty"`
}

type RequestRule struct {
	ID             string    `json:"id"`
	Method         string    `json:"method"`
	Path           string    `json:"path"`
	Conditions     Condition `json:"conditions"`
	RemediationIDs []string  `json:"doctorRemediationIds,omitempty"`
	FixIDs         []string  `json:"doctorFixIds,omitempty"`
}

type Logic struct {
	Operator string       `json:"operator"`
	Items    []Logic      `json:"items,omitempty"`
	Request  *RequestRule `json:"request,omitempty"`
}

type Catalog struct {
	ID           string        `json:"id"`
	Topic        string        `json:"topic"`
	Diagnosis    string        `json:"diagnosis"`
	Docs         []string      `json:"docs"`
	Remediations []Declaration `json:"doctorRemediations,omitempty"`
	Fixes        []LegacyFix   `json:"doctorFixes,omitempty"`
	Logic        []Logic       `json:"logic"`
	Revision     string        `json:"-"`
	Rules        []RequestRule `json:"-"`
}

type PredicateLoader interface {
	Load(context.Context, string) (Catalog, error)
}

type MockLoader struct{}

func (MockLoader) Load(_ context.Context, path string) (Catalog, error) {
	data := mockPredicates
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return Catalog{}, errors.New("predicate file unavailable")
		}
		defer f.Close()
		data, err = io.ReadAll(io.LimitReader(f, maxInput+1))
		if err != nil {
			return Catalog{}, errors.New("predicate file unreadable")
		}
	}
	return ParseCatalog(data)
}

// strictJSON rejects duplicate keys, mixed wire spellings, unknown fields and oversized trees.
func strictJSON(data []byte, out any) error {
	if len(data) > maxInput {
		return errors.New("input exceeds size budget")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	var visit func(int) error
	visit = func(depth int) error {
		if depth > 24 {
			return errors.New("input exceeds nesting budget")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		if t == nil {
			return errors.New("null is not supported; omit optional fields")
		}
		if s, ok := t.(string); ok && len(s) > 2048 {
			return errors.New("input string exceeds budget")
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				keys := map[string]bool{}
				for d.More() {
					k, err := d.Token()
					if err != nil {
						return err
					}
					key, ok := k.(string)
					if !ok || keys[key] || len(key) > 128 {
						return errors.New("duplicate or invalid JSON key")
					}
					keys[key] = true
					if err := visit(depth + 1); err != nil {
						return err
					}
				}
			case '[':
				for d.More() {
					if err := visit(depth + 1); err != nil {
						return err
					}
				}
			default:
				return errors.New("invalid JSON delimiter")
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := visit(0); err != nil {
		return errors.New("invalid or unbounded JSON")
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON input")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return errors.New("invalid JSON schema (expected documented camelCase format)")
	}
	return nil
}

func digest(data []byte) string { s := sha256.Sum256(data); return hex.EncodeToString(s[:]) }

func ParseCatalog(data []byte) (Catalog, error) {
	var c Catalog
	if err := strictJSON(data, &c); err != nil {
		return c, err
	}
	if c.ID != "legacyPaymentMethods" || c.Topic != "dpm" || c.Diagnosis == "" || len(c.Logic) == 0 {
		return c, errors.New("invalid predicate metadata")
	}
	if len(c.Remediations) > 32 || len(c.Fixes) > 32 || len(c.Docs) > 8 || len(c.Logic) > 16 {
		return c, errors.New("catalog exceeds declaration or group budget")
	}
	ids, legacy := map[string]bool{}, map[string]bool{}
	for _, d := range c.Remediations {
		if !identifier.MatchString(d.ID) || !identifier.MatchString(d.Kind) || d.Version < 1 || len(d.Options) == 0 || ids[d.ID] {
			return c, errors.New("invalid or duplicate remediation declaration")
		}
		ids[d.ID] = true
	}
	for _, f := range c.Fixes {
		if !identifier.MatchString(f.ID) || legacy[f.ID] || (f.Kind != "removeParam" && f.Kind != "upsertParam") || f.Param != "payment_method_types" {
			return c, errors.New("invalid legacy declaration")
		}
		legacy[f.ID] = true
	}
	rules := map[string]bool{}
	var walk func(Logic, int) error
	walk = func(n Logic, depth int) error {
		if depth > 8 || len(c.Rules) >= 128 {
			return errors.New("logic exceeds budget")
		}
		switch n.Operator {
		case "or":
			if n.Request != nil || len(n.Items) == 0 || len(n.Items) > 16 {
				return errors.New("invalid logic group")
			}
			for _, item := range n.Items {
				if err := walk(item, depth+1); err != nil {
					return err
				}
			}
		case "request":
			if n.Request == nil || len(n.Items) != 0 {
				return errors.New("invalid request node")
			}
			r := *n.Request
			if !identifier.MatchString(r.ID) || rules[r.ID] || r.Method != "POST" || !supportedPath(r.Path) {
				return errors.New("unsupported request method/path or duplicate rule")
			}
			rules[r.ID] = true
			if err := validateCondition(r.Conditions, 0); err != nil {
				return err
			}
			if len(r.RemediationIDs) > 8 || len(r.FixIDs) > 8 {
				return errors.New("request references exceed budget")
			}
			for _, pair := range []struct {
				refs    []string
				catalog map[string]bool
			}{{r.RemediationIDs, ids}, {r.FixIDs, legacy}} {
				seen := map[string]bool{}
				for _, id := range pair.refs {
					if !pair.catalog[id] || seen[id] {
						return errors.New("duplicate or dangling declaration reference")
					}
					seen[id] = true
				}
			}
			if r.RemediationIDs != nil && len(r.RemediationIDs) == 0 {
				return errors.New("empty new declaration reference")
			}
			c.Rules = append(c.Rules, r)
		default:
			return errors.New("unsupported logic operator")
		}
		return nil
	}
	for _, n := range c.Logic {
		if err := walk(n, 0); err != nil {
			return c, err
		}
	}
	c.Revision = digest(data)
	return c, nil
}

func supportedPath(p string) bool {
	return p == "/v1/payment_intents" || p == "/v1/setup_intents" || p == "/v1/checkout/sessions"
}

func validateCondition(c Condition, depth int) error {
	if depth > 8 {
		return errors.New("condition nesting exceeds budget")
	}
	if c.Operator == "and" || c.Operator == "or" {
		if c.Key != "" || c.Value != "" || len(c.Items) == 0 || len(c.Items) > 16 {
			return errors.New("invalid condition group")
		}
		for _, i := range c.Items {
			if err := validateCondition(i, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if len(c.Items) > 0 {
		return errors.New("invalid condition leaf")
	}
	switch c.Key {
	case "payment_method_types":
		if (c.Operator == "exists" || c.Operator == "absent") && c.Value == "" {
			return nil
		}
	case "automatic_payment_methods.enabled":
		if (c.Operator == "absent" && c.Value == "") || (c.Operator == "equals" && (c.Value == "false" || c.Value == "true")) {
			return nil
		}
	case "Stripe-Version":
		if c.Operator == "before" && c.Value == "2023-08-16" {
			return nil
		}
	}
	return errors.New("unsupported condition key/operator/value")
}

func contract(d Declaration) error {
	if d.Kind != "payment_methods.adopt_dynamic" || d.Version != 1 {
		return errors.New("unsupported handler contract; no legacy fallback")
	}
	var o struct {
		OfferingPolicy string `json:"offeringPolicy"`
	}
	if strictJSON(d.Options, &o) != nil || o.OfferingPolicy != "preserve_existing" {
		return errors.New("unsupported handler options; no legacy fallback")
	}
	return nil
}

// match returns unknown when source evidence cannot establish request semantics.
func match(c Condition, f SourceFlow) string {
	if c.Operator == "and" || c.Operator == "or" {
		unknown := false
		for _, i := range c.Items {
			v := match(i, f)
			if (c.Operator == "and" && v == "no") || (c.Operator == "or" && v == "yes") {
				return v
			}
			unknown = unknown || v == "unknown"
		}
		if unknown {
			return "unknown"
		}
		if c.Operator == "and" {
			return "yes"
		}
		return "no"
	}
	ok := false
	switch c.Key {
	case "payment_method_types":
		if f.Pinned.Status == "unknown" || f.Pinned.Status == "conditional" {
			return "unknown"
		}
		ok = (c.Operator == "exists" && f.Pinned.Status == "known") || (c.Operator == "absent" && f.Pinned.Status == "absent")
	case "automatic_payment_methods.enabled":
		if f.Automatic == "unknown" {
			return "unknown"
		}
		ok = (c.Operator == "absent" && f.Automatic == "absent") || (c.Operator == "equals" && ((c.Value == "true" && f.Automatic == "enabled") || (c.Value == "false" && f.Automatic == "disabled")))
	case "Stripe-Version":
		if len(f.Context.APIVersion) < 10 {
			return "unknown"
		}
		ok = f.Context.APIVersion[:10] < c.Value
	}
	if ok {
		return "yes"
	}
	return "no"
}
