package i18n

import (
	"context"
)

type localeKey struct{}

// WithLocale set locale to context, which will be used to match the language to the translation string.
func WithLocale(ctx context.Context, locale Tag) context.Context {
	return context.WithValue(ctx, localeKey{}, Tags{locale})
}

// WithLocales set locales to context, which will be used to match the language to the translation string.
func WithLocales(ctx context.Context, locales Tags) context.Context {
	return context.WithValue(ctx, localeKey{}, locales)
}

// GetLocales get locales from context.
func GetLocales(ctx context.Context) Tags {
	if v, ok := ctx.Value(localeKey{}).(Tags); ok {
		return v
	}
	return nil
}
