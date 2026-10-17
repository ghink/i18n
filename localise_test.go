package i18n

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// payload exercises every container kind shell must walk, in one pass.
type payload struct {
	Name   I18n            `json:"name"`
	Nested *payload        `json:"nested,omitempty"`
	List   []I18n          `json:"list"`
	Pair   [2]I18n         `json:"pair"`
	Dict   map[string]I18n `json:"dict"`
	Any    any             `json:"any"`
	Nil    *payload        `json:"nil"`
	Skips  string          `json:"skips"`
	hidden I18n            // unexported: invisible to both shell and JSON
}

func newPayload() payload {
	return payload{
		Name: I18n{"en": "Coffee", "zh": "咖啡"},
		Nested: &payload{
			Name: I18n{"en": "Bean", "zh": "豆"},
		},
		List:  []I18n{{"en": "Hot", "zh": "热"}},
		Pair:  [2]I18n{{"en": "A", "zh": "甲"}, {"en": "B", "zh": "乙"}},
		Dict:  map[string]I18n{"k": {"en": "K", "zh": "克"}},
		Any:   I18n{"en": "iface", "zh": "接口"},
		Skips: "plain",
	}
}

func TestToJSON(t *testing.T) {
	ctx := WithLocale(context.Background(), MustParseLocale("zh"))
	b, err := ToJSON(ctx, newPayload())
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}
	var got map[string]any
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "咖啡" {
		t.Errorf("name = %v, want 咖啡", got["name"])
	}
	nested, ok := got["nested"].(map[string]any)
	if !ok || nested["name"] != "豆" {
		t.Errorf("nested.name = %v, want 豆", got["nested"])
	}
	if wantList := []any{"热"}; !reflect.DeepEqual(got["list"], wantList) {
		t.Errorf("list = %v, want %v", got["list"], wantList)
	}
	pair, ok := got["pair"].([]any)
	if !ok || len(pair) != 2 || pair[0] != "甲" || pair[1] != "乙" {
		t.Errorf("pair = %v, want [甲 乙]", got["pair"])
	}
	dict, ok := got["dict"].(map[string]any)
	if !ok || dict["k"] != "克" {
		t.Errorf("dict.k = %v, want 克", got["dict"])
	}
	if got["any"] != "接口" {
		t.Errorf("any = %v, want 接口", got["any"])
	}
	if got["nil"] != nil {
		t.Errorf("nil field = %v, want null", got["nil"])
	}
	if got["skips"] != "plain" {
		t.Errorf("skips = %v, want plain", got["skips"])
	}
}

func TestToJSONNilValue(t *testing.T) {
	b, err := ToJSON(context.Background(), any(nil))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "null" {
		t.Errorf("ToJSON(nil) = %s, want null", b)
	}
}

func TestToJSONOnI18nDirectly(t *testing.T) {
	ctx := WithLocale(context.Background(), MustParseLocale("en"))
	m := I18n{"en": "Hello", "zh": "你好"}
	b, err := ToJSON(ctx, m)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"Hello"` {
		t.Errorf("ToJSON(I18n) = %s, want %q", b, `"Hello"`)
	}
}

func TestToJSONNoLocalesKeepsAllLanguages(t *testing.T) {
	// Without a locale the deterministic fallback still narrows to one entry.
	// The value is a struct, so the JSON keeps the field name; the field's
	// single-key I18n collapses to a bare string.
	b, err := ToJSON(context.Background(), struct {
		Name I18n `json:"name"`
	}{Name: I18n{"en": "x"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"name":"x"}` {
		t.Errorf("ToJSON single-key struct = %s, want %q", b, `{"name":"x"}`)
	}

	// Raw marshal (no peel) emits the full language map.
	b, err = Marshal(I18n{"en": "Hello", "zh": "你好"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "en") || !strings.Contains(string(b), "zh") {
		t.Errorf("unpeeled Marshal = %s, want full language map", b)
	}
}

func TestLocalise(t *testing.T) {
	type dto struct {
		Text string `json:"text"`
	}
	p := struct {
		Text I18n `json:"text"`
	}{Text: I18n{"en": "Hello", "zh": "你好"}}

	ctx := WithLocale(context.Background(), MustParseLocale("zh"))
	got, err := Localise[dto](ctx, p)
	if err != nil {
		t.Fatalf("Localise error: %v", err)
	}
	if got.Text != "你好" {
		t.Errorf("Localise = %+v, want Text 你好", got)
	}

	// Type mismatch between peeled output and Out must surface as error.
	type badDTO struct {
		Text int `json:"text"`
	}
	if _, err := Localise[badDTO](ctx, p); err == nil {
		t.Error("Localise into mismatched type expected error, got nil")
	}
}

func TestLocaliseKeep(t *testing.T) {
	ctx := WithLocale(context.Background(), MustParseLocale("zh"))
	orig := newPayload()
	kept := LocaliseKeep(ctx, orig)

	// Fields are narrowed to a single zh entry, type unchanged.
	if !equalI18n(kept.Name, I18n{"zh": "咖啡"}) {
		t.Errorf("kept.Name = %v, want {zh: 咖啡}", kept.Name)
	}
	if kept.Nested == nil || !equalI18n(kept.Nested.Name, I18n{"zh": "豆"}) {
		t.Errorf("kept.Nested.Name not peeled: %+v", kept.Nested)
	}
	if kept.Nested == orig.Nested {
		t.Error("LocaliseKeep must not share the nested pointer with the original")
	}

	// No locales in ctx: fallback still narrows each field to one entry.
	raw := LocaliseKeep(context.Background(), struct {
		Name I18n `json:"name"`
	}{Name: I18n{"en": "Hello"}})
	if !equalI18n(raw.Name, I18n{"en": "Hello"}) {
		t.Errorf("raw.Name = %v, want {en: Hello}", raw.Name)
	}
}

func TestLocaliseKeepInputUnchanged(t *testing.T) {
	ctx := WithLocale(context.Background(), MustParseLocale("zh"))
	orig := newPayload()
	pristine := newPayload() // independent copy with equal content

	_ = LocaliseKeep(ctx, orig)

	// Compare structurally, never by marshalled bytes: encoding/json/v2
	// deliberately randomises map key order between calls.
	if !reflect.DeepEqual(orig, pristine) {
		t.Error("input mutated by LocaliseKeep")
	}
}
