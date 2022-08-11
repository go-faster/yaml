//
// Copyright (c) 2011-2019 Canonical Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package yaml_test

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	yaml "github.com/go-faster/yamlx"
)

var marshalIntTest = 123

var marshalTests = []struct {
	value interface{}
	data  string
}{
	{
		nil,
		"null\n",
	},
	{
		(*marshalerType)(nil),
		"null\n",
	},
	{
		&struct{}{},
		"{}\n",
	},
	{
		map[string]string{"v": "hi"},
		"v: hi\n",
	},
	{
		map[string]interface{}{"v": "hi"},
		"v: hi\n",
	},
	{
		map[string]string{"v": "true"},
		"v: \"true\"\n",
	},
	{
		map[string]string{"v": "false"},
		"v: \"false\"\n",
	},
	{
		map[string]interface{}{"v": true},
		"v: true\n",
	},
	{
		map[string]interface{}{"v": false},
		"v: false\n",
	},
	{
		map[string]interface{}{"v": 10},
		"v: 10\n",
	},
	{
		map[string]interface{}{"v": -10},
		"v: -10\n",
	},
	{
		map[string]uint{"v": 42},
		"v: 42\n",
	},
	{
		map[string]interface{}{"v": int64(4294967296)},
		"v: 4294967296\n",
	},
	{
		map[string]int64{"v": int64(4294967296)},
		"v: 4294967296\n",
	},
	{
		map[string]uint64{"v": 4294967296},
		"v: 4294967296\n",
	},
	{
		map[string]interface{}{"v": "10"},
		"v: \"10\"\n",
	},
	{
		map[string]interface{}{"v": 0.1},
		"v: 0.1\n",
	},
	{
		map[string]interface{}{"v": float64(0.1)},
		"v: 0.1\n",
	},
	{
		map[string]interface{}{"v": float32(0.99)},
		"v: 0.99\n",
	},
	{
		map[string]interface{}{"v": -0.1},
		"v: -0.1\n",
	},
	{
		map[string]interface{}{"v": math.Inf(+1)},
		"v: .inf\n",
	},
	{
		map[string]interface{}{"v": math.Inf(-1)},
		"v: -.inf\n",
	},
	{
		map[string]interface{}{"v": math.NaN()},
		"v: .nan\n",
	},
	{
		map[string]interface{}{"v": nil},
		"v: null\n",
	},
	{
		map[string]interface{}{"v": ""},
		"v: \"\"\n",
	},
	{
		map[string][]string{"v": {"A", "B"}},
		"v:\n    - A\n    - B\n",
	},
	{
		map[string][]string{"v": {"A", "B\nC"}},
		"v:\n    - A\n    - |-\n        B\n        C\n",
	},
	{
		map[string][]interface{}{"v": {"A", 1, map[string][]int{"B": {2, 3}}}},
		"v:\n    - A\n    - 1\n    -   B:\n            - 2\n            - 3\n",
	},
	{
		map[string]interface{}{"a": map[interface{}]interface{}{"b": "c"}},
		"a:\n    b: c\n",
	},
	{
		map[string]interface{}{"a": "-"},
		"a: '-'\n",
	},

	// Ensure correct indentation.
	//
	// https://github.com/go-yaml/yaml/issues/643
	// https://github.com/go-faster/yamlx/issues/8
	{
		[]string{" hello\nworld"},
		"- |4-\n     hello\n    world\n",
	},
	{
		&yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{
					Kind: yaml.MappingNode,
					Tag:  "!!map",
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Tag: "!!str", Value: "a"},
						{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Style: yaml.LiteralStyle, Tag: "!!str", Value: " 00"},
						}},
					},
				},
			},
		},
		"a:\n    - |4-\n         00\n",
	},
	{
		"\t\ndetected\n",
		"|\n    \t\n    detected\n",
	},
	{
		"\tB\n\tC\n",
		"|\n    \tB\n    \tC\n",
	},
	{
		"folded line\nnext line\n * one\n * two\n\nlast line\n",
		"|\n    folded line\n    next line\n     * one\n     * two\n\n    last line\n",
	},
	{
		"\nfolded line\nnext line\n * one\n * two\n\nlast line\n",
		"|4\n\n    folded line\n    next line\n     * one\n     * two\n\n    last line\n",
	},
	{
		"# detected\n",
		"|\n    # detected\n",
	},
	{
		"\n# detected\n",
		"|4\n\n    # detected\n",
	},
	{
		"\n\n# detected\n",
		"|4\n\n\n    # detected\n",
	},
	{
		"literal\n\n\ttext\n",
		"|\n    literal\n\n    \ttext\n",
	},
	{
		"\nliteral\n\n\ttext\n",
		"|4\n\n    literal\n\n    \ttext\n",
	},
	{
		"\n\nliteral\n\n\ttext\n",
		"|4\n\n\n    literal\n\n    \ttext\n",
	},

	// Simple values.
	{
		&marshalIntTest,
		"123\n",
	},

	// Structures
	{
		&struct{ Hello string }{"world"},
		"hello: world\n",
	},
	{
		&struct {
			A struct {
				B string
			}
		}{struct{ B string }{"c"}},
		"a:\n    b: c\n",
	},
	{
		&struct {
			A *struct {
				B string
			}
		}{&struct{ B string }{"c"}},
		"a:\n    b: c\n",
	},
	{
		&struct {
			A *struct {
				B string
			}
		}{},
		"a: null\n",
	},
	{
		&struct{ A int }{1},
		"a: 1\n",
	},
	{
		&struct{ A []int }{[]int{1, 2}},
		"a:\n    - 1\n    - 2\n",
	},
	{
		&struct{ A [2]int }{[2]int{1, 2}},
		"a:\n    - 1\n    - 2\n",
	},
	{
		&struct {
			B int "a"
		}{1},
		"a: 1\n",
	},
	{
		&struct{ A bool }{true},
		"a: true\n",
	},
	{
		&struct{ A string }{"true"},
		"a: \"true\"\n",
	},
	{
		&struct{ A string }{"off"},
		"a: \"off\"\n",
	},

	// Conditional flag
	{
		&struct {
			A int "a,omitempty"
			B int "b,omitempty"
		}{1, 0},
		"a: 1\n",
	},
	{
		&struct {
			A int "a,omitempty"
			B int "b,omitempty"
		}{0, 0},
		"{}\n",
	},
	{
		&struct {
			A *struct{ X, y int } "a,omitempty,flow"
		}{&struct{ X, y int }{1, 2}},
		"a: {x: 1}\n",
	},
	{
		&struct {
			A *struct{ X, y int } "a,omitempty,flow"
		}{nil},
		"{}\n",
	},
	{
		&struct {
			A *struct{ X, y int } "a,omitempty,flow"
		}{&struct{ X, y int }{}},
		"a: {x: 0}\n",
	},
	{
		&struct {
			A struct{ X, y int } "a,omitempty,flow"
		}{struct{ X, y int }{1, 2}},
		"a: {x: 1}\n",
	},
	{
		&struct {
			A struct{ X, y int } "a,omitempty,flow"
		}{struct{ X, y int }{0, 1}},
		"{}\n",
	},
	{
		&struct {
			A float64 "a,omitempty"
			B float64 "b,omitempty"
		}{1, 0},
		"a: 1\n",
	},
	{
		&struct {
			T1 time.Time  "t1,omitempty"
			T2 time.Time  "t2,omitempty"
			T3 *time.Time "t3,omitempty"
			T4 *time.Time "t4,omitempty"
		}{
			T2: time.Date(2018, 1, 9, 10, 40, 47, 0, time.UTC),
			T4: newTime(time.Date(2098, 1, 9, 10, 40, 47, 0, time.UTC)),
		},
		"t2: 2018-01-09T10:40:47Z\nt4: 2098-01-09T10:40:47Z\n",
	},
	// Nil interface that implements Marshaler.
	{
		map[string]yaml.Marshaler{
			"a": nil,
		},
		"a: null\n",
	},

	// Flow flag
	{
		&struct {
			A []int "a,flow"
		}{[]int{1, 2}},
		"a: [1, 2]\n",
	},
	{
		&struct {
			A map[string]string "a,flow"
		}{map[string]string{"b": "c", "d": "e"}},
		"a: {b: c, d: e}\n",
	},
	{
		&struct {
			A struct {
				B, D string
			} "a,flow"
		}{struct{ B, D string }{"c", "e"}},
		"a: {b: c, d: e}\n",
	},
	{
		&struct {
			A string "a,flow"
		}{"b\nc"},
		"a: \"b\\nc\"\n",
	},

	// Unexported field
	{
		&struct {
			u int
			A int
		}{0, 1},
		"a: 1\n",
	},

	// Ignored field
	{
		&struct {
			A int
			B int "-"
		}{1, 2},
		"a: 1\n",
	},

	// Struct inlining
	{
		&struct {
			A int
			C inlineB `yaml:",inline"`
		}{1, inlineB{2, inlineC{3}}},
		"a: 1\nb: 2\nc: 3\n",
	},
	// Struct inlining as a pointer
	{
		&struct {
			A int
			C *inlineB `yaml:",inline"`
		}{1, &inlineB{2, inlineC{3}}},
		"a: 1\nb: 2\nc: 3\n",
	},
	{
		&struct {
			A int
			C *inlineB `yaml:",inline"`
		}{1, nil},
		"a: 1\n",
	},
	{
		&struct {
			A int
			D *inlineD `yaml:",inline"`
		}{1, &inlineD{&inlineC{3}, 4}},
		"a: 1\nc: 3\nd: 4\n",
	},

	// Map inlining
	{
		&struct {
			A int
			C map[string]int `yaml:",inline"`
		}{1, map[string]int{"b": 2, "c": 3}},
		"a: 1\nb: 2\nc: 3\n",
	},

	// Duration
	{
		map[string]time.Duration{"a": 3 * time.Second},
		"a: 3s\n",
	},

	// Issue #24: bug in map merging logic.
	{
		map[string]string{"a": "<foo>"},
		"a: <foo>\n",
	},

	// Issue #34: marshal unsupported base 60 floats quoted for compatibility
	// with old YAML 1.1 parsers.
	{
		map[string]string{"a": "1:1"},
		"a: \"1:1\"\n",
	},

	// Binary data.
	{
		map[string]string{"a": "\x00"},
		"a: \"\\0\"\n",
	},
	{
		map[string]string{"a": "\x80\x81\x82"},
		"a: !!binary gIGC\n",
	},
	{
		map[string]string{"a": strings.Repeat("\x90", 54)},
		"a: !!binary |\n    " + strings.Repeat("kJCQ", 17) + "kJ\n    CQ\n",
	},

	// Encode unicode as utf-8 rather than in escaped form.
	{
		map[string]string{"a": "你好"},
		"a: 你好\n",
	},
	{
		"你好",
		"你好\n",
	},

	// Support encoding.TextMarshaler.
	{
		map[string]net.IP{"a": net.IPv4(1, 2, 3, 4)},
		"a: 1.2.3.4\n",
	},
	// time.Time gets a timestamp tag.
	{
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, 0, time.UTC)},
		"a: 2015-02-24T18:19:39Z\n",
	},
	{
		map[string]*time.Time{"a": newTime(time.Date(2015, 2, 24, 18, 19, 39, 0, time.UTC))},
		"a: 2015-02-24T18:19:39Z\n",
	},
	{
		// This is confirmed to be properly decoded in Python (libyaml) without a timestamp tag.
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, 123456789, time.FixedZone("FOO", -3*60*60))},
		"a: 2015-02-24T18:19:39.123456789-03:00\n",
	},
	// Ensure timestamp-like strings are quoted.
	{
		map[string]string{"a": "2015-02-24T18:19:39Z"},
		"a: \"2015-02-24T18:19:39Z\"\n",
	},

	// Ensure strings containing ": " are quoted (reported as PR #43, but not reproducible).
	{
		map[string]string{"a": "b: c"},
		"a: 'b: c'\n",
	},

	// Containing hash mark ('#') in string should be quoted
	{
		map[string]string{"a": "Hello #comment"},
		"a: 'Hello #comment'\n",
	},
	{
		map[string]string{"a": "你好 #comment"},
		"a: '你好 #comment'\n",
	},

	// Ensure MarshalYAML also gets called on the result of MarshalYAML itself.
	{
		&marshalerType{marshalerType{true}},
		"true\n",
	},
	{
		&marshalerType{&marshalerType{true}},
		"true\n",
	},

	// Check indentation of maps inside sequences inside maps.
	{
		map[string]interface{}{"a": map[string]interface{}{"b": []map[string]int{{"c": 1, "d": 2}}}},
		"a:\n    b:\n        -   c: 1\n            d: 2\n",
	},

	// Strings with tabs were disallowed as literals (issue #471).
	{
		map[string]string{"a": "\tB\n\tC\n"},
		"a: |\n    \tB\n    \tC\n",
	},

	// Ensure that strings do not wrap
	{
		map[string]string{"a": "abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 "},
		"a: 'abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 '\n",
	},

	// yaml.Node
	{
		&struct {
			Value yaml.Node
		}{
			yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "foo",
				Style: yaml.SingleQuotedStyle,
			},
		},
		"value: 'foo'\n",
	},
	{
		yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "foo",
			Style: yaml.SingleQuotedStyle,
		},
		"'foo'\n",
	},

	// Enforced tagging with shorthand notation (issue #616).
	{
		&struct {
			Value yaml.Node
		}{
			yaml.Node{
				Kind:  yaml.ScalarNode,
				Style: yaml.TaggedStyle,
				Value: "foo",
				Tag:   "!!str",
			},
		},
		"value: !!str foo\n",
	},
	{
		&struct {
			Value yaml.Node
		}{
			yaml.Node{
				Kind:  yaml.MappingNode,
				Style: yaml.TaggedStyle,
				Tag:   "!!map",
			},
		},
		"value: !!map {}\n",
	},
	{
		&struct {
			Value yaml.Node
		}{
			yaml.Node{
				Kind:  yaml.SequenceNode,
				Style: yaml.TaggedStyle,
				Tag:   "!!seq",
			},
		},
		"value: !!seq []\n",
	},
}

func TestMarshal(t *testing.T) {
	t.Setenv("TZ", "UTC")

	for i, item := range marshalTests {
		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %#v", item.value)
			a := require.New(t)

			data, err := yaml.Marshal(item.value)
			a.NoError(err)
			a.Equal(item.data, string(data))
		})
	}
}

func TestEncoderSingleDocument(t *testing.T) {
	for i, item := range marshalTests {
		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %#v", item.value)
			a := require.New(t)

			var buf strings.Builder
			enc := yaml.NewEncoder(&buf)
			a.NoError(enc.Encode(item.value))
			a.NoError(enc.Close())
			a.Equal(item.data, buf.String())
		})
	}
}

func TestEncoderMultipleDocuments(t *testing.T) {
	a := require.New(t)

	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	a.NoError(enc.Encode(map[string]string{"a": "b"}))
	a.NoError(enc.Encode(map[string]string{"c": "d"}))
	a.NoError(enc.Close())
	a.Equal("a: b\n---\nc: d\n", buf.String())
}

func TestEncoderWriteError(t *testing.T) {
	enc := yaml.NewEncoder(errWriter{})
	err := enc.Encode(map[string]string{"a": "b"})
	require.ErrorIs(t, err, errTestWriteError)
}

type errWriter struct{}

var errTestWriteError = errors.New("some write error")

func (errWriter) Write([]byte) (int, error) {
	return 0, errTestWriteError
}

var marshalErrorTests = []struct {
	value interface{}
	error string
	panic string
}{
	{
		value: &struct {
			B       int
			inlineB ",inline"
		}{1, inlineB{2, inlineC{3}}},
		panic: `duplicated key 'b' in struct struct \{ B int; .*`,
	},
	{
		value: &struct {
			A int
			B map[string]int ",inline"
		}{1, map[string]int{"a": 2}},
		panic: `cannot have key "a" in inlined map: conflicts with struct field`,
	},
	{
		value: &yaml.Node{
			Kind: yaml.AliasNode,
		},
		error: "yaml: alias value must not be empty",
	},
	{
		value: &yaml.Node{
			Kind:  yaml.AliasNode,
			Value: "#",
		},
		error: "yaml: alias value must contain alphanumerical characters only",
	},
	{
		value: &yaml.Node{
			Kind:   yaml.ScalarNode,
			Anchor: "#",
			Value:  "10",
		},
		error: "yaml: anchor value must contain alphanumerical characters only",
	},
}

func TestMarshalErrors(t *testing.T) {
	for i, item := range marshalErrorTests {
		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			a := require.New(t)

			if item.panic != "" {
				msg := func() (s string) {
					defer func() {
						rr := recover()
						a.NotNil(rr)
						s = fmt.Sprintf("%v", rr)
					}()
					yaml.Marshal(item.value)
					return s
				}()
				a.Regexp(item.panic, msg)
				return
			}

			_, err := yaml.Marshal(item.value)
			a.Error(err)
			a.Regexp(item.error, err.Error())
		})
	}
}

func TestMarshalTypeCache(t *testing.T) {
	a := require.New(t)

	var data []byte
	var err error
	func() {
		type T struct{ A int }
		data, err = yaml.Marshal(&T{})
		a.NoError(err)
	}()
	func() {
		type T struct{ B int }
		data, err = yaml.Marshal(&T{})
		a.NoError(err)
	}()
	a.Equal("b: 0\n", string(data))
}

var marshalerTests = []struct {
	data  string
	value interface{}
}{
	{"_:\n    hi: there\n", map[interface{}]interface{}{"hi": "there"}},
	{"_:\n    - 1\n    - A\n", []interface{}{1, "A"}},
	{"_: 10\n", 10},
	{"_: null\n", nil},
	{"_: BAR!\n", "BAR!"},
}

type marshalerType struct {
	value interface{}
}

func (o marshalerType) MarshalText() ([]byte, error) {
	panic("MarshalText called on type with MarshalYAML")
}

func (o marshalerType) MarshalYAML() (interface{}, error) {
	return o.value, nil
}

type marshalerValue struct {
	Field marshalerType "_"
}

func TestMarshaler(t *testing.T) {
	for i, item := range marshalerTests {
		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %#v", item.value)
			a := require.New(t)

			obj := &marshalerValue{}
			obj.Field.value = item.value
			data, err := yaml.Marshal(obj)
			a.NoError(err)
			a.Equal(item.data, string(data))
		})
	}
}

func TestMarshalerWholeDocument(t *testing.T) {
	a := require.New(t)

	obj := &marshalerType{}
	obj.value = map[string]string{"hello": "world!"}
	data, err := yaml.Marshal(obj)
	a.NoError(err)
	a.Equal("hello: world!\n", string(data))
}

type failingMarshaler struct{}

func (ft *failingMarshaler) MarshalYAML() (interface{}, error) {
	return nil, errFailing
}

func TestMarshalerError(t *testing.T) {
	_, err := yaml.Marshal(&failingMarshaler{})
	require.ErrorIs(t, err, errFailing)
}

func TestSetIndent(t *testing.T) {
	a := require.New(t)

	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(8)
	a.NoError(enc.Encode(map[string]interface{}{"a": map[string]interface{}{"b": map[string]string{"c": "d"}}}))
	a.NoError(enc.Close())
	a.Equal("a:\n        b:\n                c: d\n", buf.String())
}

func TestSortedOutput(t *testing.T) {
	a := require.New(t)

	order := []interface{}{
		false,
		true,
		1,
		uint(1),
		1.0,
		1.1,
		1.2,
		2,
		uint(2),
		2.0,
		2.1,
		"",
		".1",
		".2",
		".a",
		"1",
		"2",
		"a!10",
		"a/0001",
		"a/002",
		"a/3",
		"a/10",
		"a/11",
		"a/0012",
		"a/100",
		"a~10",
		"ab/1",
		"b/1",
		"b/01",
		"b/2",
		"b/02",
		"b/3",
		"b/03",
		"b1",
		"b01",
		"b3",
		"c2.10",
		"c10.2",
		"d1",
		"d7",
		"d7abc",
		"d12",
		"d12a",
		"e2b",
		"e4b",
		"e21a",
	}
	m := make(map[interface{}]int)
	for _, k := range order {
		m[k] = 1
	}
	data, err := yaml.Marshal(m)
	a.NoError(err)

	out := "\n" + string(data)
	last := 0
	for i, k := range order {
		repr := fmt.Sprint(k)
		if s, ok := k.(string); ok {
			if _, err = strconv.ParseFloat(repr, 32); s == "" || err == nil {
				repr = `"` + repr + `"`
			}
		}
		a.Contains(out, "\n"+repr+":")

		index := strings.Index(out, "\n"+repr+":")
		a.NotEqual(-1, index, "%#v is not in the output: %#v", k, out)
		var prev interface{}
		if i > 0 {
			prev = order[i-1]
		}
		a.GreaterOrEqual(index, last, "%#v was generated before %#v: %q", k, prev, out)
		last = index
	}
}

func newTime(t time.Time) *time.Time {
	return &t
}

func testEncodeDecodeString(t *testing.T, input string) {
	t.Run("Scalar", func(t *testing.T) {
		defer func() {
			t.Logf("Input: %q", input)
		}()
		a := require.New(t)

		data, err := yaml.Marshal(input)
		a.NoError(err)

		defer func() {
			t.Logf("Marshal: %q", data)
		}()

		var output string
		a.NoError(yaml.Unmarshal(data, &output))
		a.Equal(input, output)
	})
	t.Run("Mapping", func(t *testing.T) {
		defer func() {
			t.Logf("Input: %q", input)
		}()
		a := require.New(t)

		input := map[string]string{"foo": input}
		data, err := yaml.Marshal(input)
		a.NoError(err)

		defer func() {
			t.Logf("Marshal: %q", data)
		}()

		var output map[string]string
		a.NoError(yaml.Unmarshal(data, &output))
		a.Equal(input, output)
	})
	t.Run("Sequence", func(t *testing.T) {
		defer func() {
			t.Logf("Input: %q", input)
		}()
		a := require.New(t)

		input := []string{input}
		data, err := yaml.Marshal(input)
		a.NoError(err)

		defer func() {
			t.Logf("Marshal: %q", data)
		}()

		var output []string
		a.NoError(yaml.Unmarshal(data, &output))
		a.Equal(input, output)
	})
}

func TestEncodeDecodeString(t *testing.T) {
	for i, tt := range []string{
		"\t\ndetected\n",
		"\tB\n\tC\n",

		"folded line\nnext line\n * one\n * two\n\nlast line\n",
		"\nfolded line\nnext line\n * one\n * two\n\nlast line\n",

		"# detected\n",
		"\n# detected\n",
		"\n\n# detected\n",

		"literal\n\n\ttext\n",
		"\nliteral\n\n\ttext\n",
		"\n\nliteral\n\n\ttext\n",
	} {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			testEncodeDecodeString(t, tt)
		})
	}
}
