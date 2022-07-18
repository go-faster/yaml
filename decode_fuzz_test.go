package yaml

import (
	"testing"
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

	f.Fuzz(func(t *testing.T, input []byte) {
		var v interface{}
		_ = Unmarshal(input, &v)
	})
}
