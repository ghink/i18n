package i18n

import (
	"context"
	"testing"

	"golang.org/x/text/language"
)

// parseTags builds Tags from locale strings; well-formed-but-unknown codes
// (e.g. "zz") are kept as root tags so the matcher treats them as no match,
// which is exactly the scenario the fallback tests need.
func parseTags(t *testing.T, locales ...string) Tags {
	t.Helper()
	tags := make(Tags, 0, len(locales))
	for _, l := range locales {
		tag, err := language.Parse(l)
		if err != nil {
			tag, _ = language.Parse("und") // root: filtered from preferences downstream
		}
		tags = append(tags, Tag(tag))
	}
	return tags
}

func TestUnwrap(t *testing.T) {
	tests := []struct {
		name    string
		m       I18n
		locales Tags
		want    I18n // nil means expect nil result
	}{
		{
			name:    "exact match",
			m:       I18n{"en": "Hello", "zh": "你好"},
			locales: MustParseLocales([]string{"zh"}),
			want:    I18n{"zh": "你好"},
		},
		{
			name:    "matcher negotiation: zh-HK falls back to zh",
			m:       I18n{"en": "Hello", "zh": "你好"},
			locales: MustParseLocales([]string{"zh-HK"}),
			want:    I18n{"zh": "你好"},
		},
		{
			name:    "matcher negotiation: en-US climbs to en",
			m:       I18n{"en": "Hello", "zh": "你好"},
			locales: MustParseLocales([]string{"en-US"}),
			want:    I18n{"en": "Hello"},
		},
		{
			name:    "preference order wins among equals",
			m:       I18n{"en": "Hello", "fr": "Bonjour"},
			locales: MustParseLocales([]string{"fr", "en"}),
			want:    I18n{"fr": "Bonjour"},
		},
		{
			name:    "no match: lexicographically first non-empty wins",
			m:       I18n{"ja": "こんにちは", "zh": "你好"},
			locales: MustParseLocales([]string{"ko"}),
			want:    I18n{"ja": "こんにちは"},
		},
		{
			name:    "fallback skips empty values",
			m:       I18n{"a": "", "b": "Bee"},
			locales: parseTags(t, "zz"), // zz is well-formed but unknown; matcher sees No
			want:    I18n{"b": "Bee"},
		},
		{
			name:    "no locales: deterministic fallback",
			m:       I18n{"zh": "你好", "en": "Hello"},
			locales: nil,
			want:    I18n{"en": "Hello"},
		},
		{
			name:    "invalid keys are ignored for matching but kept for fallback",
			m:       I18n{"klingon!": "ghobe'", "en": "Hello"},
			locales: MustParseLocales([]string{"en"}),
			want:    I18n{"en": "Hello"},
		},
		{
			name:    "root preference (und) is filtered out",
			m:       I18n{"und": "Und", "en": "Hello"},
			locales: Tags{MustParseLocale("und"), MustParseLocale("en")},
			want:    I18n{"en": "Hello"},
		},
		{
			name:    "all values empty yields nil",
			m:       I18n{"en": "", "  ": ""},
			locales: MustParseLocales([]string{"en"}),
			want:    nil,
		},
		{
			name:    "empty map yields nil",
			m:       I18n{},
			locales: MustParseLocales([]string{"en"}),
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.Unwrap(tt.locales)
			if len(got) != len(tt.want) {
				t.Fatalf("Unwrap = %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("Unwrap = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// TestUnwrapTieIsDeterministic guards a real bug: candidate order used to come
// straight from map iteration, and the matcher breaks equal scores by position
// (earlier candidate wins), so zh vs zh-CN flipped results between runs. With
// candidates sorted lexicographically the generic zh now wins — one unified
// rule with the Unwrap fallback: smallest key takes ties. Many iterations make
// randomised map order statistically impossible to pass by luck.
func TestUnwrapTieIsDeterministic(t *testing.T) {
	m := I18n{"zh": "你好", "zh-CN": "你好呀", "en": "Hello"}
	locales := MustParseLocales([]string{"zh"})

	for i := range 200 {
		if got := m.Pick(locales); got != "你好" {
			t.Fatalf("iteration %d: Pick(zh) = %q, want %q (tie won by lexicographically smallest key)", i, got, "你好")
		}
	}

	// The same must hold through context-based resolution.
	ctx := WithLocale(context.Background(), MustParseLocale("zh"))
	for i := range 200 {
		if got := m.PickCtx(ctx); got != "你好" {
			t.Fatalf("iteration %d: PickCtx = %q, want %q", i, got, "你好")
		}
	}

	// An exact zh-CN preference still beats the generic zh entry.
	if got := m.Pick(MustParseLocales([]string{"zh-CN"})); got != "你好呀" {
		t.Errorf("Pick(zh-CN) = %q, want %q", got, "你好呀")
	}
}

func TestUnwrapDeduplicatesEquivalentKeys(t *testing.T) {
	// "EN" and "en" normalise to the same tag; duplicates must not confuse
	// the matcher. Which original key survives is unspecified, the value is not.
	got := I18n{"EN": "Hello", "en": "Hello"}.Unwrap(MustParseLocales([]string{"en"}))
	if len(got) != 1 {
		t.Fatalf("Unwrap = %v, want exactly one entry", got)
	}
	for _, v := range got {
		if v != "Hello" {
			t.Errorf("Unwrap value = %q, want %q", v, "Hello")
		}
	}
}

func TestPick(t *testing.T) {
	m := I18n{"en": "Hello", "zh": "你好"}
	if got := m.Pick(MustParseLocales([]string{"zh"})); got != "你好" {
		t.Errorf("Pick(zh) = %q, want %q", got, "你好")
	}
	if got := (I18n{}).Pick(MustParseLocales([]string{"en"})); got != "" {
		t.Errorf("Pick on empty map = %q, want empty string", got)
	}
}

func TestCtxVariants(t *testing.T) {
	base := context.Background()
	ctx := WithLocale(base, MustParseLocale("fr"))
	m := I18n{"en": "Hello", "fr": "Bonjour"}

	if got := m.PickCtx(ctx); got != "Bonjour" {
		t.Errorf("PickCtx = %q, want %q", got, "Bonjour")
	}
	if got := m.UnwrapCtx(ctx); !equalI18n(got, I18n{"fr": "Bonjour"}) {
		t.Errorf("UnwrapCtx = %v, want {%q: %q}", got, "fr", "Bonjour")
	}

	// GetLocales on a context without locale returns nil.
	if got := GetLocales(base); got != nil {
		t.Errorf("GetLocales(empty ctx) = %v, want nil", got)
	}
	// WithLocales overwrites a previous single locale.
	ctx2 := WithLocales(ctx, MustParseLocales([]string{"en"}))
	if got := m.PickCtx(ctx2); got != "Hello" {
		t.Errorf("PickCtx after WithLocales = %q, want %q", got, "Hello")
	}
}

func equalI18n(a, b I18n) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
