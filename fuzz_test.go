package yaml_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	yaml "github.com/go-faster/yamlx"
)

var _ testingF = (*testing.F)(nil)

type testingF interface {
	Add(args ...interface{})
}

func addFuzzingCorpus(f testingF) {
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
}

func FuzzDecodeEncodeDecode(f *testing.F) {
	addFuzzingCorpus(f)

	f.Fuzz(func(t *testing.T, input []byte) {
		defer func() {
			r := recover()
			if r != nil || t.Failed() || t.Skipped() {
				t.Logf("Input: %q", input)
			}
		}()

		var v yaml.Node
		if err := yaml.Unmarshal(input, &v); err != nil {
			t.Skipf("Error: %+v", err)
			return
		}

		a := require.New(t)
		data, err := yaml.Marshal(&v)
		a.NoError(err)

		var v2 yaml.Node
		a.NoError(yaml.Unmarshal(data, &v2), "Data: %q", data)

		if v.IsZero() != v2.IsZero() {
			t.Logf("v.IsZero() != v2.IsZero(), %v != %v", v.IsZero(), v2.IsZero())
			t.Skipf("Zero value, data: %q", data)
			return
		}

		var compareNodes func(n1, n2 *yaml.Node)
		compareNodes = func(n1, n2 *yaml.Node) {
			a.Equal(n1.ShortTag(), n2.ShortTag())
			a.Equal(n1.Value, n2.Value)
			a.Equal(len(n1.Content), len(n2.Content))
			for i := range n1.Content {
				compareNodes(n1.Content[i], n2.Content[i])
			}
		}
		compareNodes(&v, &v2)
	})
}
