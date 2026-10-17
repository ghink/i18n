package i18n

import (
	"context"
	"reflect"
)

var i18nType = reflect.TypeFor[I18n]()

// shell deep copy + peel off, original v unchanged.
func shell(v reflect.Value, locales Tags) reflect.Value {
	if !v.IsValid() {
		return v
	}
	if v.Type() == i18nType {
		u := v.Interface().(I18n).Unwrap(locales)
		if u == nil {
			return reflect.Zero(v.Type())
		}
		return reflect.ValueOf(u)
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		n := reflect.New(v.Type().Elem())
		n.Elem().Set(shell(v.Elem(), locales))
		return n
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		n := reflect.New(v.Type()).Elem()
		n.Set(shell(v.Elem(), locales))
		return n
	case reflect.Struct:
		n := reflect.New(v.Type()).Elem()
		n.Set(v)
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !t.Field(i).IsExported() {
				continue
			}
			if f := n.Field(i); f.CanSet() {
				f.Set(shell(v.Field(i), locales))
			}
		}
		return n
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		n := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			n.Index(i).Set(shell(v.Index(i), locales))
		}
		return n
	case reflect.Array:
		n := reflect.New(v.Type()).Elem()
		for i := 0; i < v.Len(); i++ {
			n.Index(i).Set(shell(v.Index(i), locales))
		}
		return n
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		n := reflect.MakeMapWithSize(v.Type(), v.Len())
		for _, k := range v.MapKeys() {
			n.SetMapIndex(k, shell(v.MapIndex(k), locales))
		}
		return n
	default:
		return v
	}
}

// ToJSON directly outputs JSON after unwrapping
func ToJSON(ctx context.Context, v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	shelled := shell(reflect.ValueOf(v), GetLocales(ctx))
	return Marshal(shelled.Interface())
}

// Localise de-shells and converts to Out. Out is specified by the caller
func Localise[Out any](ctx context.Context, v any) (Out, error) {
	var out Out
	b, err := ToJSON(ctx, v)
	if err != nil {
		return out, err
	}
	if err = Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}

// LocaliseKeep keeps the original type and returns a de-shelled copy,
// fields are still I18n (single key)
func LocaliseKeep[T any](ctx context.Context, v T) T {
	var out T
	reflect.ValueOf(&out).Elem().Set(shell(reflect.ValueOf(v), GetLocales(ctx)))
	return out
}
