package i18n

import (
	"reflect"
	"strings"
	"testing"

	"golang.org/x/text/language"
)

// tagStrs renders Tags as their canonical strings for readable assertions.
func tagStrs(tags Tags) []string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = language.Tag(t).String()
	}
	return out
}

func TestParseLocale(t *testing.T) {
	tag, err := ParseLocale("zh-CN")
	if err != nil {
		t.Fatalf("ParseLocale(zh-CN) error: %v", err)
	}
	if got := language.Tag(tag).String(); got != "zh-CN" {
		t.Errorf("ParseLocale(zh-CN) = %q, want %q", got, "zh-CN")
	}

	if _, err := ParseLocale("not a locale!"); err == nil {
		t.Error("ParseLocale(invalid) expected error, got nil")
	}
}

func TestMustParseLocale(t *testing.T) {
	if got := language.Tag(MustParseLocale("fr")).String(); got != "fr" {
		t.Errorf("MustParseLocale(fr) = %q, want %q", got, "fr")
	}

	defer func() {
		if recover() == nil {
			t.Error("MustParseLocale(invalid) expected panic")
		}
	}()
	MustParseLocale("not a locale!")
}

func TestParseLocales(t *testing.T) {
	tags, err := ParseLocales([]string{"en", "zh-CN"})
	if err != nil {
		t.Fatalf("ParseLocales error: %v", err)
	}
	if want := []string{"en", "zh-CN"}; !reflect.DeepEqual(tagStrs(tags), want) {
		t.Errorf("ParseLocales = %v, want %v", tagStrs(tags), want)
	}

	if _, err := ParseLocales([]string{"en", "not a locale!"}); err == nil {
		t.Error("ParseLocales(invalid entry) expected error, got nil")
	}
}

func TestMustParseLocales(t *testing.T) {
	tags := MustParseLocales([]string{"en", "zh"})
	if want := []string{"en", "zh"}; !reflect.DeepEqual(tagStrs(tags), want) {
		t.Errorf("MustParseLocales = %v, want %v", tagStrs(tags), want)
	}

	defer func() {
		if recover() == nil {
			t.Error("MustParseLocales(invalid) expected panic")
		}
	}()
	MustParseLocales([]string{"en", "not a locale!"})
}

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   []string
	}{
		{
			name:   "sorted by descending weight",
			header: "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7",
			want:   []string{"en-US", "en", "zh-CN", "zh"},
		},
		{
			name:   "missing q defaults to highest",
			header: "fr",
			want:   []string{"fr"},
		},
		{
			name:   "whitespace and unordered weights",
			header: "de;q=0.5, es;q=0.9",
			want:   []string{"es", "de"},
		},
		{
			name:   "zero weight entries are dropped",
			header: "en;q=0, ja",
			want:   []string{"ja"},
		},
		{
			name:   "wildcard maps to mul",
			header: "*",
			want:   []string{"mul"},
		},
		{
			name:   "empty header yields no tags",
			header: "",
			want:   nil,
		},
		{
			name:   "bare comma is tolerated and yields no tags",
			header: ",",
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags, err := ParseAcceptLanguage(tt.header)
			if err != nil {
				t.Fatalf("ParseAcceptLanguage(%q) error: %v", tt.header, err)
			}
			if !reflect.DeepEqual(tagStrs(tags), tt.want) && !(len(tags) == 0 && len(tt.want) == 0) {
				t.Errorf("ParseAcceptLanguage(%q) = %v, want %v", tt.header, tagStrs(tags), tt.want)
			}
		})
	}
}

func TestParseAcceptLanguageError(t *testing.T) {
	// Not well-formed ranges make the whole header fail.
	if _, err := ParseAcceptLanguage("garbage,,x"); err == nil {
		t.Error("ParseAcceptLanguage(garbage,,x) expected error, got nil")
	}
}

func TestMustParseAcceptLanguage(t *testing.T) {
	tags := MustParseAcceptLanguage("zh-CN,zh;q=0.9")
	if got, want := tagStrs(tags), []string{"zh-CN", "zh"}; !reflect.DeepEqual(got, want) {
		t.Errorf("MustParseAcceptLanguage = %v, want %v", got, want)
	}

	defer func() {
		if recover() == nil {
			t.Error("MustParseAcceptLanguage(invalid) expected panic")
		}
	}()
	MustParseAcceptLanguage("garbage,,x")
}

func TestParseAcceptLanguageWithLocales(t *testing.T) {
	// End-to-end: header weights feed the matcher. zh (q=0.9, exact match)
	// outranks zh-CN (q=0.5); even with equal scores the lexicographically
	// smallest key would win, per the rule pinned in TestUnwrapTieIsDeterministic.
	i := I18n{"en": "Hello", "zh": "你好", "zh-CN": "你好呀"}
	tags, err := ParseAcceptLanguage("zh-CN;q=0.5,zh;q=0.9")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tagStrs(tags), ","), "zh") {
		t.Fatalf("unexpected tags: %v", tags)
	}
	if got := i.Pick(tags); got != "你好" {
		t.Errorf("Pick with header-derived locales = %q, want %q", got, "你好")
	}
}
