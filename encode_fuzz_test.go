package yaml_test

import (
	"testing"
	"unicode/utf8"
)

func FuzzEncodeDecodeString(f *testing.F) {
	var collectStrings func(f *testing.F, input any)
	collectStrings = func(f *testing.F, input any) {
		switch input := input.(type) {
		case nil:
			return
		case string:
			if utf8.ValidString(input) {
				f.Add(input)
			}
		case map[string]any:
			for _, v := range input {
				collectStrings(f, v)
			}
		case map[string]string:
			for _, v := range input {
				collectStrings(f, v)
			}
		case []any:
			for _, v := range input {
				collectStrings(f, v)
			}
		case []string:
			for _, v := range input {
				collectStrings(f, v)
			}
		}
	}

	collectStrings(f, encodeDecodeStringTests)
	for _, tt := range marshalTests {
		collectStrings(f, tt.value)
	}
	for _, tt := range unmarshalTests {
		collectStrings(f, tt.value)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skipf("Invalid UTF8 string: %q", input)
			return
		}
		testEncodeDecodeString(t, input)
	})
}
