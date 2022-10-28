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
	"encoding"
	"fmt"
	"math"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-faster/errors"

	yaml "github.com/go-faster/yamlx"
)

var marshalIntTest = 123

var marshalTests = []struct {
	value any
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
		map[string]any{"v": "hi"},
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
		map[string]any{"v": true},
		"v: true\n",
	},
	{
		map[string]any{"v": false},
		"v: false\n",
	},
	{
		map[string]any{"v": 10},
		"v: 10\n",
	},
	{
		map[string]any{"v": -10},
		"v: -10\n",
	},
	{
		map[string]uint{"v": 42},
		"v: 42\n",
	},
	{
		map[string]any{"v": int64(4294967296)},
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
		map[string]any{"v": "10"},
		"v: \"10\"\n",
	},
	{
		map[string]any{"v": 0.1},
		"v: 0.1\n",
	},
	{
		map[string]any{"v": float64(0.1)},
		"v: 0.1\n",
	},
	{
		map[string]any{"v": float32(0.99)},
		"v: 0.99\n",
	},
	{
		map[string]any{"v": -0.1},
		"v: -0.1\n",
	},
	{
		map[string]any{"v": math.Inf(+1)},
		"v: .inf\n",
	},
	{
		map[string]any{"v": math.Inf(-1)},
		"v: -.inf\n",
	},
	{
		map[string]any{"v": math.NaN()},
		"v: .nan\n",
	},
	{
		map[string]any{"v": nil},
		"v: null\n",
	},
	{
		map[string]any{"v": ptrTo[any](nil)},
		"v: null\n",
	},
	{
		map[string]any{"v": ""},
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
		map[string][]any{"v": {"A", 1, map[string][]int{"B": {2, 3}}}},
		"v:\n    - A\n    - 1\n    -   B:\n            - 2\n            - 3\n",
	},
	{
		map[string]any{"a": map[any]any{"b": "c"}},
		"a:\n    b: c\n",
	},
	{
		map[string]any{"a": "-"},
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
	// https://github.com/go-yaml/yaml/issues/804.
	{
		yaml.Node{
			Kind:  yaml.ScalarNode,
			Style: yaml.FoldedStyle,
			Value: "foo\n  bar",
			Tag:   "!!str",
		},
		">-\n    foo\n      bar\n",
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
		&struct{ A [2]int }{},
		"a:\n    - 0\n    - 0\n",
	},
	{
		&struct {
			A [2]int `yaml:"a,omitempty"`
			B int    `yaml:"b"`
		}{},
		"b: 0\n",
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
			T4: ptrTo(time.Date(2098, 1, 9, 10, 40, 47, 0, time.UTC)),
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
	{
		map[string]encoding.TextMarshaler{
			"a": nil,
		},
		"a: null\n",
	},

	// Map of any type.
	//
	// https://github.com/go-yaml/yaml/issues/912
	{
		map[string]any{
			"a": time.Date(2018, 1, 9, 10, 40, 47, 0, time.UTC),
			"b": ptrTo(time.Date(2098, 1, 9, 10, 40, 47, 0, time.UTC)),
			"c": (*time.Time)(nil),
		},
		"a: 2018-01-09T10:40:47Z\nb: 2098-01-09T10:40:47Z\nc: null\n",
	},
	{
		map[string]any{
			// *any -> time.Time
			"a": ptrTo[any](time.Date(2018, 1, 9, 10, 40, 47, 0, time.UTC)),
			// *any -> *time.Time
			"b": ptrTo[any](ptrTo(time.Date(2098, 1, 9, 10, 40, 47, 0, time.UTC))),
		},
		"a: 2018-01-09T10:40:47Z\nb: 2098-01-09T10:40:47Z\n",
	},
	{
		map[string]any{
			"a": yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "foo",
			},
			"b": &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "bar",
			},
			"c": (*yaml.Node)(nil),
		},
		"a: foo\nb: bar\nc: null\n",
	},
	{
		map[string]any{
			"a": time.Second,
			"b": ptrTo[time.Duration](time.Second),
			"c": (*time.Duration)(nil),
		},
		"a: 1s\nb: 1s\nc: null\n",
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
	//
	// See https://github.com/go-yaml/yaml/issues/737.
	{
		map[string]string{"a": "你好"},
		"a: 你好\n",
	},
	{
		"你好",
		"你好\n",
	},
	{
		map[string]string{"a": "🛑"},
		"a: 🛑\n",
	},
	// Notice that result is not escaped.
	{
		map[string]string{"a": "\U0001f3f3\ufe0f\u200d\U0001f308"},
		"a: " + "\U0001f3f3\ufe0f\u200d\U0001f308" + "\n",
	},
	{"\U0001f3f3\ufe0f\u200d\U0001f308", "\U0001f3f3\ufe0f\u200d\U0001f308\n"},
	{"\U0001f439", "\U0001f439\n"},
	{"\U0001f1fa\U0001f1f8", "\U0001f1fa\U0001f1f8\n"},
	{"\U0001f474\U0001f3ff", "\U0001f474\U0001f3ff\n"},

	// Anchor names.
	{
		yaml.Node{
			Kind:   yaml.ScalarNode,
			Anchor: "the_\U0001f3f3\ufe0f\u200d\U0001f308_anchor",
			Value:  "10",
		},
		"&the_\U0001f3f3\ufe0f\u200d\U0001f308_anchor 10\n",
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
		map[string]*time.Time{"a": ptrTo(time.Date(2015, 2, 24, 18, 19, 39, 0, time.UTC))},
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
		map[string]any{"a": map[string]any{"b": []map[string]int{{"c": 1, "d": 2}}}},
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

	// Special tag cases.
	{
		yaml.Node{
			Kind: yaml.ScalarNode,
			Tag:  "!!",
		},
		"!<!!>\n",
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

const invalidUTF8String = "\xff"

var marshalErrorTests = []struct {
	value any
	error string
	panic string
}{
	// Inline struct field conflicts with other struct field.
	//
	// Notice that we check for the panic message here, not the error.
	{
		value: &struct {
			B       int
			inlineB ",inline"
		}{1, inlineB{2, inlineC{3}}},
		panic: `duplicated key 'b' in struct struct \{ B int; .*`,
	},

	// Inline map key conflicts with struct field.
	//
	// Notice that we check error message here, not the panic. It's because
	// key may be generated at run time, whereas struct tag is defined at
	// compile time.
	{
		value: &struct {
			A int
			B map[string]int ",inline"
		}{1, map[string]int{"a": 2}},
		error: `cannot have key "a" in inlined map: conflicts with struct field A`,
	},

	// Tagged string marshaling.
	{
		value: &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: invalidUTF8String,
		},
		error: "yaml: cannot marshal invalid UTF-8 data as !!str",
	},
	{
		value: &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!binary",
			Value: invalidUTF8String,
		},
		error: "yaml: explicitly tagged !!binary data must be base64-encoded",
	},

	// Alias and anchor marshaling.
	{
		value: &yaml.Node{
			Kind: yaml.AliasNode,
		},
		error: "yaml: alias value must not be empty",
	},
	{
		value: &yaml.Node{
			Kind:  yaml.AliasNode,
			Value: ",",
		},
		error: "yaml: alias value must contain alphanumerical characters only",
	},
	{
		value: &yaml.Node{
			Kind:   yaml.ScalarNode,
			Anchor: ",",
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
	value any
}{
	{"_:\n    hi: there\n", map[any]any{"hi": "there"}},
	{"_:\n    - 1\n    - A\n", []any{1, "A"}},
	{"_: 10\n", 10},
	{"_: null\n", nil},
	{"_: BAR!\n", "BAR!"},
}

type marshalerType struct {
	value any
}

func (o marshalerType) MarshalText() ([]byte, error) {
	panic("MarshalText called on type with MarshalYAML")
}

func (o marshalerType) MarshalYAML() (any, error) {
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

var _ yaml.Marshaler = (*failingMarshaler)(nil)

type failingMarshaler struct{}

func (ft *failingMarshaler) MarshalYAML() (any, error) {
	return nil, errFailing
}

func TestMarshalerError(t *testing.T) {
	_, err := yaml.Marshal(&failingMarshaler{})
	require.ErrorIs(t, err, errFailing)
}

var _ encoding.TextMarshaler = (*failingTextMarshaler)(nil)

type failingTextMarshaler struct{}

func (ft *failingTextMarshaler) MarshalText() ([]byte, error) {
	return nil, errFailing
}

func TestTextMarshalerError(t *testing.T) {
	_, err := yaml.Marshal(&failingTextMarshaler{})
	require.ErrorIs(t, err, errFailing)
}

func TestSetIndent(t *testing.T) {
	a := require.New(t)

	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(8)
	a.NoError(enc.Encode(map[string]any{"a": map[string]any{"b": map[string]string{"c": "d"}}}))
	a.NoError(enc.Close())
	a.Equal("a:\n        b:\n                c: d\n", buf.String())
}

func TestSortedOutput(t *testing.T) {
	a := require.New(t)

	order := []any{
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
	m := make(map[any]int)
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
		var prev any
		if i > 0 {
			prev = order[i-1]
		}
		a.GreaterOrEqual(index, last, "%#v was generated before %#v: %q", k, prev, out)
		last = index
	}
}

func ptrTo[T any](val T) *T {
	return &val
}

func testEncodeDecodeString(t *testing.T, input string) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			name  string
			input any
		}{
			{
				"Scalar",
				input,
			},
			{
				"Mapping",
				map[string]string{"foo": input},
			},
			{
				"Sequence",
				[]string{input},
			},
		}
		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				defer func() {
					if r := recover(); t.Failed() || r != nil {
						t.Logf("Input: %#v", tt.input)
					}
				}()
				a := require.New(t)

				data, err := yaml.Marshal(tt.input)
				a.NoError(err)

				defer func() {
					if r := recover(); t.Failed() || r != nil {
						t.Logf("Marshal: %q", data)
					}
				}()

				typ := reflect.TypeOf(tt.input)
				target := reflect.New(typ)
				a.NoError(yaml.Unmarshal(data, target.Interface()))

				output := target.Elem().Interface()
				a.Equal(tt.input, output)
			})
		}
	})
	t.Run("Node", func(t *testing.T) {
		for _, style := range []yaml.Style{
			0,
			yaml.DoubleQuotedStyle,
			yaml.SingleQuotedStyle,
			yaml.LiteralStyle,
			yaml.FoldedStyle,
		} {
			tt := struct {
				input yaml.Node
			}{
				input: yaml.Node{
					Kind:  yaml.ScalarNode,
					Style: style,
					Value: input,
				},
			}
			t.Run(fmt.Sprintf("%sStyle", style), func(t *testing.T) {
				defer func() {
					if r := recover(); t.Failed() || r != nil {
						t.Logf("Input: %#v", tt.input)
					}
				}()
				a := require.New(t)

				data, err := yaml.Marshal(tt.input)
				a.NoError(err)

				defer func() {
					if r := recover(); t.Failed() || r != nil {
						t.Logf("Marshal: %q", data)
					}
				}()

				var output yaml.Node
				a.NoError(yaml.Unmarshal(data, &output))
				if output.Kind == yaml.DocumentNode {
					output = *output.Content[0]
				}
				a.Equal(tt.input.Value, output.Value)
			})
		}
	})
}

var encodeDecodeStringTests = []string{
	"",

	// Control characters.
	"\x00",
	"\x01",
	"\a",
	"\b",
	"\t",
	"\n",
	"\v",
	"\f", "\f\f",
	"\r",
	"\x1a",
	"\u00a0",
	"\u001b",
	" ",
	"\u007F",
	"\u0085",
	"\u009F",
	"\u2028",
	"\u2029",
	"\uFEFF",
	"\uFFF9",
	"\uFFFA",
	"\uFFFB",

	// Special characters.
	"\"", "'", "`",
	"#", "# #", "\n# #",
	":", ";", ",",
	".", "...", "....",
	">", ">>", ">>>",
	"?", "!", "!!", "!!str",
	"[", "]", "[]", "[0]",
	"{", "}", "{}", "{0:0}",
	"(", ")",
	"\\", "\\\\",
	"|",
	"&", "&&", "&foo", "&amp;",
	"*", "**", "*foo",
	"%", "%%", "%20", "%aa",
	"-", "---", "----",
	"@", "$", "~", "+", "_",

	// Numbers.
	"0",
	"-0",
	"0.1",
	"-0.1",
	"0e1",
	"-0e1",
	"0..1",
	"100",
	"0b1",
	"-0b1",
	"01",
	"-01",
	"0o1",
	"-0o1",
	"0xff",
	"-0xff",

	// Some plain cases.
	"foo",
	"foo\n",
	"\nfoo",
	"\tfoo",
	" foo",
	"\n\nfoo",
	"\n\tfoo",
	"# foo",
	"\n# foo",
	"foo\"",
	"- foo\n - bar\n",

	// Unicode cases.
	"\u00FF", // Max Latin-1.
	"щ",
	"сша",
	"你",
	// Emoji.
	"\U0001f439",
	"\U0001f1fa\U0001f1f8",
	"\U0001f474\U0001f3ff",
	"\U0001f3f3\ufe0f\u200d\U0001f308",

	// Test cases from original yaml.v3 and YAML suite.
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

	// Found by fuzzer.
	"0\n0",
	"0\n\n0",
	"0\n\n\n0",
}

func TestEncodeDecodeString(t *testing.T) {
	for i, tt := range encodeDecodeStringTests {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			testEncodeDecodeString(t, tt)
		})
	}
}
