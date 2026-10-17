# i18n

Easy i18n for Go. [中文文档](README.zh-CN.md)

A tiny internationalisation library that lets you store translations **inline** right next
to your data. Instead of pulling strings from external `.po`/`.json` catalogues at runtime,
you keep a multi-language map on each field and let the library pick the best match for the
current locale when you serialise or read the value.

- Store translations as a simple `map[languageCode]text`.
- Carry the user's locale (preference list) in a `context.Context`.
- Recursively "peel" any struct, slice, map or pointer down to a single language.
- Powered by [`golang.org/x/text/language`](https://pkg.go.dev/golang.org/x/text/language)
  for real best-match locale negotiation, with a deterministic fallback.

## Requirements

- Go **1.27+**.

Install:

```sh
go get go.gh.ink/i18n
```

## Core type

```go
// I18n is the basic type for internationalisation:
// the key is the language code, the value is the translation.
type I18n map[string]string
```

A language code is any tag understood by `golang.org/x/text/language`, e.g. `en`, `zh`,
`zh-CN`, `fr`, `pt-BR`.

```go
title := i18n.I18n{
    "en":    "Hello",
    "zh-CN": "你好",
    "fr":    "Bonjour",
}
```

Locale tags themselves are represented by `Tag` / `Tags`:

```go
type Tag language.Tag // a single BCP-47 language tag
type Tags = []Tag     // an ordered preference list, most preferred first
```

## Reading a single value: `Unwrap` / `Pick`

Given a preference list, resolve a map down to one entry:

```go
locales := i18n.MustParseLocales([]string{"zh-CN", "en"})

title.Unwrap(locales) // i18n.I18n{"zh-CN": "你好"} — still an I18n, one key
title.Pick(locales)   // "你好"                     — the plain string
```

`Unwrap` returns a **single-entry** `I18n` (or `nil` if the map is empty / has no
non-empty value), while `Pick` returns the **string** directly. There are `*Ctx` variants
that read the locale list from a context:

```go
title.UnwrapCtx(ctx) // i18n.I18n
title.PickCtx(ctx)   // string
```

## Carrying locale in a context

The locale is stored in the context so it can travel through your handlers and reach the
serialisation helpers:

```go
// A single preferred locale.
ctx := i18n.WithLocale(context.Background(), i18n.MustParseLocale("zh-CN"))

// A full preference list (most preferred first).
ctx = i18n.WithLocales(ctx, i18n.MustParseLocales([]string{"zh-CN", "en"}))

// Read it back.
locales := i18n.GetLocales(ctx) // i18n.Tags
```

### Parsing helpers

| Function | Behaviour |
| --- | --- |
| `ParseLocale(string) (Tag, error)` | Parse one locale string. |
| `MustParseLocale(string) Tag` | Parse one locale, panics on error. |
| `ParseLocales([]string) (Tags, error)` | Parse a list of locales. |
| `MustParseLocales([]string) Tags` | Parse a list, panics on error. |
| `ParseAcceptLanguage(string) (Tags, error)` | Parse an HTTP `Accept-Language` header, honouring `q` weights. |
| `MustParseAcceptLanguage(string) Tags` | Same, panics on error. |

For HTTP handlers, `ParseAcceptLanguage` turns the `Accept-Language` header into a preference
list. It is a thin wrapper over [`language.ParseAcceptLanguage`](https://pkg.go.dev/golang.org/x/text/language#ParseAcceptLanguage):
entries are sorted by descending `q` weight (a missing `q` defaults to `1`), zero-weight
ranges are dropped, and ties keep their first-occurrence order. It returns an error for an
empty or fully malformed header, so fall back to a default:

```go
locales, err := i18n.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
if err != nil || len(locales) == 0 {
    locales = i18n.MustParseLocales([]string{"en"}) // sensible default
}
ctx := i18n.WithLocales(r.Context(), locales)
```

For example, `zh-CN,zh;q=0.9,en;q=0.8` yields the preference order `zh-CN → zh → en`.

## Localising whole structs

Declare your data with `I18n` fields and the library walks the value for you. It handles
nested structs, pointers, slices, arrays, maps and interfaces — any exported field whose
type is `I18n` gets peeled to the best matching language.

```go
type Product struct {
    Name   i18n.I18n   `json:"name"`
    Desc   i18n.I18n   `json:"desc"`
    Labels []i18n.I18n `json:"labels"`
}

p := Product{
    Name: i18n.I18n{"en": "Coffee", "zh-CN": "咖啡"},
    Desc: i18n.I18n{"en": "Hot drink", "zh-CN": "热饮"},
    Labels: []i18n.I18n{
        {"en": "Vegan", "zh-CN": "纯素"},
    },
}
```

Three entry points, all driven by the locales stored in `ctx`:

### `ToJSON(ctx, v) ([]byte, error)`

Peel every `I18n` field to a single language and marshal to JSON in one step. Because a
single-language value serialises as a plain string (see [JSON encoding](#json-encoding)),
the output contains readable text rather than language maps:

```go
ctx := i18n.WithLocale(context.Background(), i18n.MustParseLocale("zh-CN"))
b, err := i18n.ToJSON(ctx, p)
// {"name":"咖啡","desc":"热饮","labels":["纯素"]}
```

### `Localise[Out](ctx, v) (Out, error)`

Same peel + marshal, then decode into a **different** type you choose. Handy for turning a
model that carries `I18n` fields into a flat DTO of plain strings:

```go
type ProductDTO struct {
    Name   string   `json:"name"`
    Desc   string   `json:"desc"`
    Labels []string `json:"labels"`
}

dto, err := i18n.Localise[ProductDTO](ctx, p)
// ProductDTO{Name: "咖啡", Desc: "热饮", Labels: []string{"纯素"}}
```

### `LocaliseKeep[T](ctx, v T) T`

Return a **copy of the same type** whose `I18n` fields have been peeled to a single key.
The original value is left untouched, and the fields stay `I18n` (still compatible with a
later `Pick`), rather than collapsing to `string`:

```go
kept := i18n.LocaliseKeep(ctx, p) // still a Product
kept.Name.Pick(nil)               // "咖啡" — already narrowed to one language
```

> `ToJSON` / `Localise` collapse `I18n` to strings for the wire; `LocaliseKeep` keeps the
> `I18n` type so the value can be re-resolved later. All three are non-mutating: they work
> on a deep copy and never modify the input.

## JSON encoding

`I18n` implements `MarshalJSON` / `UnmarshalJSON` so it round-trips naturally, based on how
many entries it holds:

| Value | Encodes to |
| --- | --- |
| `I18n{}` (empty) | `""` |
| `I18n{"en": "Hello"}` (single) | `"Hello"` |
| `I18n{"en": "Hi", "zh": "你好"}` (multi) | `{"en":"Hi","zh":"你好"}` |

Decoding is symmetric:

| JSON | Decodes to |
| --- | --- |
| `null` / empty | `nil` |
| `"你好"` (a string) | `I18n{"und": "你好"}` (`und` = undetermined) |
| `{"en":"Hi"}` (an object) | `I18n{"en": "Hi"}` |

This is why a fully localised struct serialises to plain strings: after peeling, each field
holds exactly one entry.

## Matching and fallback

When resolving a map for a preference list, `i18n`:

1. Builds the set of **candidate** keys from entries whose value is non-empty and whose key
   is a valid language tag (duplicates removed).
2. Drops root / `und` (`Tag{}`) entries from the requested preferences.
3. Uses `language.NewMatcher(...).Match(preferences...)` to pick the best match — this gives
   real negotiation, so a request for `zh-HK` can still fall back to `zh`, etc.
4. When several candidates score equally (e.g. both `zh` and `zh-CN` satisfy a `zh` request),
   the lexicographically smallest key wins, so the outcome never depends on map iteration
   order.
5. If nothing matches (`language.No`), falls back to the **lexicographically smallest key
   with a non-empty value**, so the result stays deterministic regardless of map iteration
   order.

If the map is empty, or has no non-empty value, `Unwrap` returns `nil` and `Pick` returns
`""`.

## Complete example

```go
package main

import (
    "context"
    "fmt"

    "go.gh.ink/i18n"
)

type Greeting struct {
    Text i18n.I18n `json:"text"`
}

func main() {
    ctx := i18n.WithLocales(
        context.Background(),
        i18n.MustParseLocales([]string{"fr", "en"}),
    )

    g := Greeting{Text: i18n.I18n{"en": "Hello", "fr": "Bonjour"}}

    b, _ := i18n.ToJSON(ctx, g)
    fmt.Println(string(b)) // {"text":"Bonjour"}

    fmt.Println(g.Text.PickCtx(ctx)) // Bonjour
}
```

## API reference

| Symbol | Description |
| --- | --- |
| `type I18n map[string]string` | The translation map. |
| `type Tag` / `type Tags` | Locale tag / ordered preference list. |
| `WithLocale` / `WithLocales` / `GetLocales` | Read & write locale in a context. |
| `ParseLocale` / `MustParseLocale` / `ParseLocales` / `MustParseLocales` | Parse locale strings. |
| `ParseAcceptLanguage` / `MustParseAcceptLanguage` | Parse an `Accept-Language` header, honouring `q` weights. |
| `I18n.Unwrap` / `UnwrapCtx` | Resolve to a single-entry `I18n`. |
| `I18n.Pick` / `PickCtx` | Resolve to a plain `string`. |
| `ToJSON` | Peel + marshal to JSON. |
| `Localise[Out]` | Peel + decode into a target type. |
| `LocaliseKeep[T]` | Peel into a copy of the same type. |
| `Marshal` / `Unmarshal` | JSON codecs (`encoding/json/v2`). |

## License

Licensed under the [Apache License 2.0](LICENSE).
