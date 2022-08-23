package yaml_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	yaml "github.com/go-faster/yamlx"
)

func addFuzzingCorpus(add func(data []byte)) {
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

		"scalar: >\n next\n line\n  * one\n",
		// https://github.com/go-faster/yamlx/issues/8
		"0:\n    #00\n    - |1 \n      00",
		// https://github.com/go-faster/yamlx/pull/19#issuecomment-1221479649
		"|+\n\n#00000",
	}

	for _, data := range cases {
		add([]byte(data))
	}

	for _, item := range unmarshalTests {
		add([]byte(item.data))
	}
	for _, item := range unmarshalerTests {
		add([]byte(item.data))
	}
	add([]byte(mergeTests))
	for _, item := range unmarshalStrictTests {
		add([]byte(item.data))
	}

	for _, item := range marshalTests {
		add([]byte(item.data))
	}
	for _, item := range marshalerTests {
		add([]byte(item.data))
	}
}

func FuzzDecodeEncodeDecode(f *testing.F) {
	add := func(data []byte) {
		var n yaml.Node
		if err := yaml.Unmarshal(data, &n); err == nil {
			f.Add(data)
		}
	}
	addFuzzingCorpus(add)
	for _, tt := range readJSONSuite(f) {
		if tt.Action == Accept {
			add(tt.Data)
		}
	}
	compareTags, _ := strconv.ParseBool(os.Getenv("YAMLX_FUZZ_COMPARE_TAGS"))

	f.Fuzz(func(t *testing.T, input []byte) {
		var (
			data []byte
			err  error
		)
		defer func() {
			r := recover()
			if r != nil || t.Failed() || t.Skipped() {
				t.Logf("Input: %q", input)
				if data != nil {
					t.Logf("Data: %q", data)
				}
			}
		}()

		var v yaml.Node
		if err := yaml.Unmarshal(input, &v); err != nil {
			t.Skipf("Error: %+v", err)
			return
		}
		if v.Kind == yaml.DocumentNode {
			// FIXME(tdakkota): parser/scanner thinks that comments are part of children nodes.
			v.HeadComment = ""
			v.LineComment = ""
			v.FootComment = ""
		}

		a := require.New(t)
		data, err = yaml.Marshal(&v)
		a.NoError(err)

		var v2 yaml.Node
		a.NoError(yaml.Unmarshal(data, &v2))

		if v.IsZero() != v2.IsZero() {
			t.Skipf("v.IsZero() != v2.IsZero(), %v != %v", v.IsZero(), v2.IsZero())
			return
		}

		var compareNodes func(n1, n2 *yaml.Node)
		compareNodes = func(n1, n2 *yaml.Node) {
			a.Equal(n1.Kind, n2.Kind)
			if compareTags {
				a.Equal(n1.ShortTag(), n2.ShortTag())
			}
			a.Equal(n1.Value, n2.Value)

			// Compare aliases and anchors.
			a.Equal(n1.Anchor, n2.Anchor)
			if n1.Alias == nil {
				// Ensure that n2.Alias is nil as well.
				a.Nil(n2.Alias)
			} else {
				a.NotNil(n2.Alias)
				compareNodes(n1.Alias, n2.Alias)
			}

			// Compare children.
			a.Equal(len(n1.Content), len(n2.Content))
			for i := range n1.Content {
				compareNodes(n1.Content[i], n2.Content[i])
			}
		}
		compareNodes(&v, &v2)
	})
}
