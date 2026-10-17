package i18n

import (
	"context"
	"reflect"
	"testing"
)

func TestWithLocale(t *testing.T) {
	tag := MustParseLocale("zh-CN")
	ctx := WithLocale(context.Background(), tag)

	got := GetLocales(ctx)
	if want := (Tags{tag}); !reflect.DeepEqual(got, want) {
		t.Errorf("GetLocales after WithLocale = %v, want %v", got, want)
	}
}

func TestWithLocales(t *testing.T) {
	tags := MustParseLocales([]string{"fr", "en"})
	ctx := WithLocales(context.Background(), tags)

	got := GetLocales(ctx)
	if !reflect.DeepEqual(got, tags) {
		t.Fatalf("GetLocales after WithLocales = %v, want %v", got, tags)
	}

	// Current behaviour, pinned deliberately: WithLocales stores the caller's
	// slice without copying, so context and caller share the same backing
	// array. If isolation is ever needed, copy on the way in — not out — so
	// repeated GetLocales calls stay consistent.
	got[0] = Tag{}
	if &tags[0] != &got[0] {
		t.Fatal("WithLocales/GetLocales no longer share the caller's backing array; update this test together with the implementation")
	}
	tags[0] = MustParseLocale("fr")
	if reflect.DeepEqual(GetLocales(ctx), []Tag{{}}) {
		t.Error("GetLocales returned a stale copy; expected shared backing array")
	}
}

func TestGetLocalesAbsent(t *testing.T) {
	if got := GetLocales(context.Background()); got != nil {
		t.Errorf("GetLocales on plain context = %v, want nil", got)
	}
	// A foreign value under an unrelated key must not leak out.
	ctx := context.WithValue(context.Background(), localeKey{}, "not tags")
	if got := GetLocales(ctx); got != nil {
		t.Errorf("GetLocales with wrong stored type = %v, want nil", got)
	}
}
