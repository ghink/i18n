package i18n

import "golang.org/x/text/language"

// ParseLocale parses a locale string into a Tag.
func ParseLocale(locale string) (Tag, error) {
	tag, err := language.Parse(locale)
	if err != nil {
		return Tag{}, err
	}
	return Tag(tag), nil
}

// MustParseLocale parses a locale string into a Tag, panicking if the locale is invalid.
func MustParseLocale(locale string) Tag {
	return Tag(language.MustParse(locale))
}

// ParseLocales parses a slice of locale strings into Tags.
func ParseLocales(locales []string) (Tags, error) {
	var tags Tags
	for _, locale := range locales {
		tag, err := language.Parse(locale)
		if err != nil {
			return Tags{}, err
		}
		tags = append(tags, Tag(tag))
	}
	return tags, nil
}

// MustParseLocales parses a slice of locale strings into Tags, panicking if any locale is invalid.
func MustParseLocales(locales []string) Tags {
	var tags Tags
	for _, locale := range locales {
		tags = append(tags, Tag(language.MustParse(locale)))
	}
	return tags
}

// ParseAcceptLanguage parses the contents of an HTTP Accept-Language header.
func ParseAcceptLanguage(header string) (Tags, error) {
	ts, _, err := language.ParseAcceptLanguage(header)
	if err != nil {
		return nil, err
	}
	tags := make(Tags, len(ts))
	for i, t := range ts {
		tags[i] = Tag(t)
	}
	return tags, nil
}

// MustParseAcceptLanguage is like ParseAcceptLanguage but panics on error.
func MustParseAcceptLanguage(header string) Tags {
	tags, err := ParseAcceptLanguage(header)
	if err != nil {
		panic(err)
	}
	return tags
}
