package yaml_test

import (
	"testing"

	yaml "github.com/go-faster/yamlx"
)

func FuzzUnmarshal(f *testing.F) {
	cases := []string{
		// runtime error: index out of range
		"\"\\0\\\r\n",

		// should not happen
		"  0: [\n] 0",
		"? ? \"\n\" 0",
		"    - {\n000}0",
		"0:\n  0: [0\n] 0",
		"    - \"\n000\"0",
		"    - \"\n000\"\"",
		"0:\n    - {\n000}0",
		"0:\n    - \"\n000\"0",
		"0:\n    - \"\n000\"\"",

		// runtime error: index out of range
		" \ufeff\n",
		"? \ufeff\n",
		"? \ufeff:\n",
		"0: \ufeff\n",
		"? \ufeff: \ufeff\n",
	}

	for _, data := range cases {
		f.Add([]byte(data))
	}

	for _, item := range unmarshalTests {
		f.Add([]byte(item.data))
	}
	for _, item := range unmarshalerTests {
		f.Add([]byte(item.data))
	}
	f.Add([]byte(mergeTests))
	for _, item := range unmarshalStrictTests {
		f.Add([]byte(item.data))
	}

	for _, item := range marshalTests {
		f.Add([]byte(item.data))
	}
	for _, item := range marshalerTests {
		f.Add([]byte(item.data))
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		var v interface{}
		_ = yaml.Unmarshal(input, &v)
	})
}
