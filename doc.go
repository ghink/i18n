// Package i18n provides easy internationalisation for Go.
//
// Design: translations are data, not catalogue strings. An I18n value is a
// plain map from language code to text, carried inline on your own structs,
// typically loaded straight from a database column:
//
//	type Article struct {
//		Title i18n.I18n `json:"title"` // {"en":"Hello","zh":"你好"}
//	}
//
// Locale is passed through a context.Context — WithLocale/WithLocales when
// entering a request, GetLocales when resolving — so the preference list
// travels with the request instead of threading through call signatures:
//
//	ctx := i18n.WithLocales(r.Context(), i18n.MustParseAcceptLanguage(r.Header.Get("Accept-Language")))
//
// When outputting, the structure is walked recursively (pointers, structs,
// slices, arrays, maps, interfaces) and every I18n field is peeled down to a
// single best-matching language. Negotiation is delegated to
// golang.org/x/text/language, so zh-HK falls back to zh, en-US climbs to en,
// and so on. Ties between equally good candidates (zh vs zh-CN for a zh
// request) and the no-match fallback are both resolved by lexicographically
// smallest key, keeping output deterministic regardless of map iteration order.
//
// Three entry points cover the common needs:
//
//   - ToJSON(ctx, v)        — peel and marshal in one step, for HTTP responses.
//   - Localise[Out](ctx, v) — peel and decode into a plain-string DTO type.
//   - LocaliseKeep[T](ctx, v) — peel into a copy of the same type; fields stay
//     I18n values narrowed to one language and can be re-resolved later.
//
// JSON encoding is deliberately asymmetric, mirroring the two directions of a
// real API: an empty map encodes as "", a single-entry map as its bare string,
// and a multi-entry map as the full language object. Decoding a bare string
// stores it under the "und" (undetermined) key. So localised output collapses
// to plain text, while an admin submitting either {"en":"..."} or "..." both
// round-trip into the same type.
//
// Note that an I18n value which is marshalled without being peeled first (for
// example a raw multi-language map that no locale matched) serialises as the
// full language object. Use ToJSON/Localise/LocaliseKeep for user-facing
// output.
//
// Scope: this package targets content translations stored with your data. It
// does not provide plural rules, interpolation, or catalogue-style message
// keys; locale negotiation itself is delegated to golang.org/x/text/language.
package i18n
