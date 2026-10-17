package i18n

import (
	"context"
	"sort"
	"strings"

	"golang.org/x/text/language"
)

// Unwrap from locales take i18n.
func (m I18n) Unwrap(locales Tags) I18n {
	if len(m) == 0 {
		return nil
	}
	if k, ok := m.bestKey(locales); ok {
		return I18n{k: m[k]}
	}

	// fallback: take the dictionary order first non-empty value, ensure determinism
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if strings.TrimSpace(v) != "" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	return I18n{keys[0]: m[keys[0]]}
}

// UnwrapCtx from context take locales and then unwrap i18n.
func (m I18n) UnwrapCtx(ctx context.Context) I18n {
	return m.Unwrap(GetLocales(ctx))
}

// Pick from locales take value.
func (m I18n) Pick(locales Tags) string {
	for _, v := range m.Unwrap(locales) {
		return v
	}
	return ""
}

// PickCtx from context take locales and then take value.
func (m I18n) PickCtx(ctx context.Context) string {
	return m.Pick(GetLocales(ctx))
}

// bestKey find best key from m using language.Matcher
func (m I18n) bestKey(locales Tags) (string, bool) {
	if len(locales) == 0 {
		return "", false
	}

	// Collect valid keys and remove duplicates,
	// NewMatcher behaviour is not guaranteed for duplicate tags
	type entry struct {
		key string
		tag language.Tag
	}
	var entries []entry
	seen := make(map[language.Tag]bool)
	for k, v := range m {
		if strings.TrimSpace(v) == "" {
			continue
		}
		t, err := language.Parse(k)
		if err != nil {
			continue
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		entries = append(entries, entry{k, t})
	}
	if len(entries) == 0 {
		return "", false
	}

	tags := make([]language.Tag, len(entries))
	for i, e := range entries {
		tags[i] = e.tag
	}

	// Filter out und (zero value / Tag{})
	pref := make([]language.Tag, 0, len(locales))
	for _, t := range locales {
		lt := language.Tag(t)
		if lt.IsRoot() {
			continue
		}
		pref = append(pref, lt)
	}
	if len(pref) == 0 {
		return "", false
	}

	matcher := language.NewMatcher(tags)
	_, idx, conf := matcher.Match(pref...)
	if conf == language.No || idx < 0 || idx >= len(entries) {
		return "", false
	}
	return entries[idx].key, true
}
