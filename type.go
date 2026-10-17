package i18n

import (
	"golang.org/x/text/language"
)

// I18n is a basic type for internationalisation,
// the key is the language code, and the value is the translation string.
type I18n map[string]string

// Tag is a type for language locale,
// which is used to match the language to the translation string.
type Tag language.Tag
type Tags = []Tag
