package i18n

import (
	"reflect"
	"testing"
)

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   I18n
		want string
	}{
		{"nil map encodes as empty string", nil, `""`},
		{"empty map encodes as empty string", I18n{}, `""`},
		{"single entry collapses to bare string", I18n{"en": "Hello"}, `"Hello"`},
		{"single entry with empty value", I18n{"en": ""}, `""`},
		{"multi entry keeps the full language map", I18n{"en": "Hi", "zh": "你好"}, `{"en":"Hi","zh":"你好"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := Marshal(tt.in)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			// Map key order is unspecified for multi-entry; compare decoded.
			if len(tt.in) > 1 {
				var got, want map[string]string
				if err := Unmarshal(b, &got); err != nil {
					t.Fatal(err)
				}
				if err := Unmarshal([]byte(tt.want), &want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("Marshal = %s, want %s", b, tt.want)
				}
				return
			}
			if string(b) != tt.want {
				t.Errorf("Marshal = %s, want %s", b, tt.want)
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  I18n
	}{
		{"null decodes to nil", `null`, nil},
		{"bare string is stored under und", `"你好"`, I18n{"und": "你好"}},
		{"empty string decodes to empty map", `""`, I18n{"und": ""}},
		{"object decodes to map", `{"en":"Hi","zh":"你好"}`, I18n{"en": "Hi", "zh": "你好"}},
		{"empty object decodes to empty map", `{}`, I18n{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got I18n
			if err := Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("Unmarshal(%s) error: %v", tt.input, err)
			}
			if !equalI18n(got, tt.want) && !(len(got) == 0 && len(tt.want) == 0) {
				t.Errorf("Unmarshal(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnmarshalJSONError(t *testing.T) {
	var got I18n
	if err := Unmarshal([]byte(`{"en": 1}`), &got); err == nil {
		t.Error("Unmarshal with wrong value type expected error, got nil")
	}
}

func TestRoundTrip(t *testing.T) {
	// Multi-language map survives a full encode/decode cycle.
	orig := I18n{"en": "Hello", "zh": "你好"}
	b, err := Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}
	var back I18n
	if err := Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !equalI18n(orig, back) {
		t.Errorf("round trip = %v, want %v", back, orig)
	}

	// Peeled single value survives as a bare string and comes back as und.
	peeled := orig.Unwrap(MustParseLocales([]string{"zh"}))
	b, err = Marshal(peeled)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"你好"` {
		t.Fatalf("Marshal(peeled) = %s, want %q", b, `"你好"`)
	}
	var s string
	if err := Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	if s != "你好" {
		t.Errorf("decoded string = %q, want %q", s, "你好")
	}
}
