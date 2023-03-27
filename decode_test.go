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
	"bytes"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-faster/errors"

	"github.com/go-faster/yaml"
)

var unmarshalIntTest = 123

var unmarshalTests = []struct {
	data  string
	value any
}{
	{
		"",
		(*struct{})(nil),
	},
	{
		"{}", &struct{}{},
	},
	{
		"v: hi",
		map[string]string{"v": "hi"},
	},
	{
		"v: hi", map[string]any{"v": "hi"},
	},
	{
		"v: true",
		map[string]string{"v": "true"},
	},
	{
		"v: true",
		map[string]any{"v": true},
	},
	{
		"v: 10",
		map[string]any{"v": 10},
	},
	{
		"v: 0b10",
		map[string]any{"v": 2},
	},
	{
		"v: 0xA",
		map[string]any{"v": 10},
	},
	{
		"v: 4294967296",
		map[string]int64{"v": 4294967296},
	},
	{
		"v: 0.1",
		map[string]any{"v": 0.1},
	},
	{
		"v: .1",
		map[string]any{"v": 0.1},
	},
	{
		"v: .Inf",
		map[string]any{"v": math.Inf(+1)},
	},
	{
		"v: -.Inf",
		map[string]any{"v": math.Inf(-1)},
	},
	{
		"v: -10",
		map[string]any{"v": -10},
	},
	{
		"v: -.1",
		map[string]any{"v": -0.1},
	},

	// Simple values.
	{
		"123",
		&unmarshalIntTest,
	},

	// Floats from spec
	{
		"canonical: 6.8523e+5",
		map[string]any{"canonical": 6.8523e+5},
	},
	{
		"expo: 685.230_15e+03",
		map[string]any{"expo": 685.23015e+03},
	},
	{
		"fixed: 685_230.15",
		map[string]any{"fixed": 685230.15},
	},
	{
		"neginf: -.inf",
		map[string]any{"neginf": math.Inf(-1)},
	},
	{
		"fixed: 685_230.15",
		map[string]float64{"fixed": 685230.15},
	},
	// {"sexa: 190:20:30.15", map[string]interface{}{"sexa": 0}}, // Unsupported
	// {"notanum: .NaN", map[string]interface{}{"notanum": math.NaN()}}, // Equality of NaN fails.

	// Bools are per 1.2 spec.
	{
		"canonical: true",
		map[string]any{"canonical": true},
	},
	{
		"canonical: false",
		map[string]any{"canonical": false},
	},
	{
		"bool: True",
		map[string]any{"bool": true},
	},
	{
		"bool: False",
		map[string]any{"bool": false},
	},
	{
		"bool: TRUE",
		map[string]any{"bool": true},
	},
	{
		"bool: FALSE",
		map[string]any{"bool": false},
	},
	// For backwards compatibility with 1.1, decoding old strings into typed values still works.
	{
		"option: on",
		map[string]bool{"option": true},
	},
	{
		"option: y",
		map[string]bool{"option": true},
	},
	{
		"option: Off",
		map[string]bool{"option": false},
	},
	{
		"option: No",
		map[string]bool{"option": false},
	},
	{
		"option: other",
		map[string]bool{},
	},
	// Ints from spec
	{
		"canonical: 685230",
		map[string]any{"canonical": 685230},
	},
	{
		"decimal: +685_230",
		map[string]any{"decimal": 685230},
	},
	{
		"octal: 02472256",
		map[string]any{"octal": 685230},
	},
	{
		"octal: -02472256",
		map[string]any{"octal": -685230},
	},
	{
		"octal: 0o2472256",
		map[string]any{"octal": 685230},
	},
	{
		"octal: -0o2472256",
		map[string]any{"octal": -685230},
	},
	{
		"hexa: 0x_0A_74_AE",
		map[string]any{"hexa": 685230},
	},
	{
		"bin: 0b1010_0111_0100_1010_1110",
		map[string]any{"bin": 685230},
	},
	{
		"bin: -0b101010",
		map[string]any{"bin": -42},
	},
	{
		"bin: -0b1000000000000000000000000000000000000000000000000000000000000000",
		map[string]any{"bin": func() any {
			val := int64(-9223372036854775808)
			if strconv.IntSize == 64 {
				return int(val)
			}
			return val
		}()},
	},
	{
		"decimal: +685_230",
		map[string]int{"decimal": 685230},
	},

	// {"sexa: 190:20:30", map[string]interface{}{"sexa": 0}}, // Unsupported

	// Nulls from spec
	{
		"empty:",
		map[string]any{"empty": nil},
	},
	{
		"canonical: ~",
		map[string]any{"canonical": nil},
	},
	{
		"english: null",
		map[string]any{"english": nil},
	},
	{
		"~: null key",
		map[any]string{nil: "null key"},
	},
	{
		"empty:",
		map[string]*bool{"empty": nil},
	},

	// Flow sequence
	{
		"seq: [A,B]",
		map[string]any{"seq": []any{"A", "B"}},
	},
	{
		"seq: [A,B,C,]",
		map[string][]string{"seq": {"A", "B", "C"}},
	},
	{
		"seq: [A,1,C]",
		map[string][]string{"seq": {"A", "1", "C"}},
	},
	{
		"seq: [A,1,C]",
		map[string][]int{"seq": {1}},
	},
	{
		"seq: [A,1,C]",
		map[string]any{"seq": []any{"A", 1, "C"}},
	},
	// Question marks in plain scalars in flow collections
	//
	// https://github.com/yaml/yaml-test-suite/issues/62.
	// https://github.com/yaml/libyaml/pull/105.
	{
		"- [a?string]",
		[]any{
			[]any{
				"a?string",
			},
		},
	},
	{

		"- a?string\n- another ? string\n- key: value?\n- [a?string]\n- [another ? string]\n- {key: value? }\n- {key: value?}\n- {key?: value }\n",
		[]any{
			"a?string",
			"another ? string",
			map[string]any{
				"key": "value?",
			},
			[]any{
				"a?string",
			},
			[]any{
				"another ? string",
			},
			map[string]any{
				"key": "value?",
			},
			map[string]any{
				"key": "value?",
			},
			map[string]any{
				"key?": "value",
			},
		},
	},
	// https://github.com/yaml/libyaml/pull/104.
	{
		"- [\"http://foo\"]",
		[]any{
			[]any{"http://foo"},
		},
	},
	{
		"- { \"foo::\": bar }",
		[]any{
			map[string]any{"foo::": "bar"},
		},
	},
	{
		"- [ \":foo\" ]",
		[]any{
			[]any{":foo"},
		},
	},
	{
		"- [ \"foo:\" ]",
		[]any{
			[]any{"foo:"},
		},
	},

	// Block sequence
	{
		"seq:\n - A\n - B",
		map[string]any{"seq": []any{"A", "B"}},
	},
	{
		"seq:\n - A\n - B\n - C",
		map[string][]string{"seq": {"A", "B", "C"}},
	},
	{
		"seq:\n - A\n - 1\n - C",
		map[string][]string{"seq": {"A", "1", "C"}},
	},
	{
		"seq:\n - A\n - 1\n - C",
		map[string][]int{"seq": {1}},
	},
	{
		"seq:\n - A\n - 1\n - C",
		map[string]any{"seq": []any{"A", 1, "C"}},
	},

	// Literal block scalar
	{
		"scalar: | # Comment\n\n literal\n\n \ttext\n\n",
		map[string]string{"scalar": "\nliteral\n\n\ttext\n"},
	},

	// Folded block scalar
	{
		"scalar: > # Comment\n\n folded\n line\n \n next\n line\n  * one\n  * two\n\n last\n line\n\n",
		map[string]string{"scalar": "\nfolded line\nnext line\n * one\n * two\n\nlast line\n"},
	},
	{
		"- >\n \t\n detected\n",
		[]string{"\t\ndetected\n"},
	},
	{
		"- >\n \t\n \t\t\t\n detected\n",
		[]string{"\t\n\t\t\t\ndetected\n"},
	},

	// Map inside interface with no type hints.
	{
		"a: {b: c}",
		map[any]any{"a": map[string]any{"b": "c"}},
	},
	// Non-string map inside interface with no type hints.
	{
		"a: {b: c, 1: d}",
		map[any]any{"a": map[any]any{"b": "c", 1: "d"}},
	},

	// Structs and type conversions.
	{
		"hello: world",
		&struct{ Hello string }{"world"},
	},
	{
		"a: {b: c}",
		&struct{ A struct{ B string } }{struct{ B string }{"c"}},
	},
	{
		"a: {b: c}",
		&struct{ A *struct{ B string } }{&struct{ B string }{"c"}},
	},
	{
		"a: 'null'",
		&struct{ A *unmarshalerType }{&unmarshalerType{"null"}},
	},
	{
		"a: {b: c}",
		&struct{ A map[string]string }{map[string]string{"b": "c"}},
	},
	{
		"a: {b: c}",
		&struct{ A *map[string]string }{&map[string]string{"b": "c"}},
	},
	{
		"a:",
		&struct{ A map[string]string }{},
	},
	{
		"a: 1",
		&struct{ A int }{1},
	},
	{
		"a: 1",
		&struct{ A float64 }{1},
	},
	{
		"a: 1.0",
		&struct{ A int }{1},
	},
	{
		"a: 1.0",
		&struct{ A uint }{1},
	},
	{
		"a: [1, 2]",
		&struct{ A []int }{[]int{1, 2}},
	},
	{
		"a: [1, 2]",
		&struct{ A [2]int }{[2]int{1, 2}},
	},
	{
		"a: 1",
		&struct{ B int }{0},
	},
	{
		"a: 1",
		&struct {
			B int "a"
		}{1},
	},
	{
		// Some limited backwards compatibility with the 1.1 spec.
		"a: YES",
		&struct{ A bool }{true},
	},

	// Some cross type conversions
	{
		"v: 42",
		map[string]uint{"v": 42},
	},
	{
		"v: -42",
		map[string]uint{},
	},
	{
		"v: 4294967296",
		map[string]uint64{"v": 4294967296},
	},
	{
		"v: -4294967296",
		map[string]uint64{},
	},

	// int
	{
		"int_max: 2147483647",
		map[string]int{"int_max": math.MaxInt32},
	},
	{
		"int_min: -2147483648",
		map[string]int{"int_min": math.MinInt32},
	},
	{
		"int_overflow: 9223372036854775808", // math.MaxInt64 + 1
		map[string]int{},
	},

	// int64
	{
		"int64_max: 9223372036854775807",
		map[string]int64{"int64_max": math.MaxInt64},
	},
	{
		"int64_max_base2: 0b111111111111111111111111111111111111111111111111111111111111111",
		map[string]int64{"int64_max_base2": math.MaxInt64},
	},
	{
		"int64_min: -9223372036854775808",
		map[string]int64{"int64_min": math.MinInt64},
	},
	{
		"int64_neg_base2: -0b111111111111111111111111111111111111111111111111111111111111111",
		map[string]int64{"int64_neg_base2": -math.MaxInt64},
	},
	{
		"int64_overflow: 9223372036854775808", // math.MaxInt64 + 1
		map[string]int64{},
	},

	// uint
	{
		"uint_min: 0",
		map[string]uint{"uint_min": 0},
	},
	{
		"uint_max: 4294967295",
		map[string]uint{"uint_max": math.MaxUint32},
	},
	{
		"uint_underflow: -1",
		map[string]uint{},
	},

	// uint64
	{
		"uint64_min: 0",
		map[string]uint{"uint64_min": 0},
	},
	{
		"uint64_max: 18446744073709551615",
		map[string]uint64{"uint64_max": math.MaxUint64},
	},
	{
		"uint64_max_base2: 0b1111111111111111111111111111111111111111111111111111111111111111",
		map[string]uint64{"uint64_max_base2": math.MaxUint64},
	},
	{
		"uint64_maxint64: 9223372036854775807",
		map[string]uint64{"uint64_maxint64": math.MaxInt64},
	},
	{
		"uint64_underflow: -1",
		map[string]uint64{},
	},

	// float32
	{
		"float32_max: 3.40282346638528859811704183484516925440e+38",
		map[string]float32{"float32_max": math.MaxFloat32},
	},
	{
		"float32_nonzero: 1.401298464324817070923729583289916131280e-45",
		map[string]float32{"float32_nonzero": math.SmallestNonzeroFloat32},
	},
	{
		"float32_maxuint64: 18446744073709551615",
		map[string]float32{"float32_maxuint64": float32(math.MaxUint64)},
	},
	{
		"float32_maxuint64+1: 18446744073709551616",
		map[string]float32{"float32_maxuint64+1": float32(math.MaxUint64 + 1)},
	},

	// float64
	{
		"float64_max: 1.797693134862315708145274237317043567981e+308",
		map[string]float64{"float64_max": math.MaxFloat64},
	},
	{
		"float64_nonzero: 4.940656458412465441765687928682213723651e-324",
		map[string]float64{"float64_nonzero": math.SmallestNonzeroFloat64},
	},
	{
		"float64_maxuint64: 18446744073709551615",
		map[string]float64{"float64_maxuint64": float64(math.MaxUint64)},
	},
	{
		"float64_maxuint64+1: 18446744073709551616",
		map[string]float64{"float64_maxuint64+1": float64(math.MaxUint64 + 1)},
	},

	// Overflow cases.
	{
		"v: 4294967297",
		map[string]int32{},
	},
	{
		"v: 128",
		map[string]int8{},
	},

	// Quoted values.
	{
		"'1': '\"2\"'",
		map[any]any{"1": "\"2\""},
	},
	{
		"v:\n- A\n- 'B\n\n  C'\n",
		map[string][]string{"v": {"A", "B\nC"}},
	},
	{
		"escaped slash: \"a\\/b\"",
		map[any]any{"escaped slash": "a/b"},
	},
	// Quoted, with escaped surrogate pairs.
	//
	// See https://github.com/go-yaml/yaml/issues/279.
	{`"\x41 \u0041 \U00000041"`, "A A A"},
	// Encode as surrogate pair \u, with upper and case lower hexadecimal digits.
	{`"\ud83d\ude04"`, "😄"},
	{`"\ud83D\udE04"`, "😄"},
	{`"\uD83D\uDE04"`, "😄"},
	// Encode as one \U.
	{`"\U0001f604"`, "😄"},
	{`"\U0001F604"`, "😄"},
	// Encode as surrogate pair \U.
	{`"\U0000D83D\U0000DE04"`, "😄"},
	{`"\uD83D\uDE04\uD83D\uDE04"`, "😄😄"},
	{`"\U0001F604\U0001F604"`, "😄😄"},
	{`"\U0000D83D\U0000DE04\U0000D83D\U0000DE04"`, "😄😄"},
	{`"_\uD83D\uDE04_\uD83D\uDE04_"`, "_😄_😄_"},
	{`"_\U0000D83D\U0000DE04_\U0000D83D\U0000DE04_"`, "_😄_😄_"},
	{`"_\U0001F604_\U0001F604_"`, "_😄_😄_"},
	{`"\u4e2d\u6587"`, "中文"},
	{`"\U00004E2D\U00006587"`, "中文"},
	{`"\ud83c\udff3\ufe0f\u200d\ud83c\udf08"`, "🏳️\u200d🌈"},
	// Test emoji handling.
	{`"Iñtërnâtiônàlizætiøn,💝🐹🌇⛔"`, "Iñtërnâtiônàlizætiøn,💝🐹🌇⛔"},
	{`"_😄_😄_"`, "_😄_😄_"},
	{`_😄_😄_`, "_😄_😄_"},
	{`"中文"`, "中文"},
	{`中文`, "中文"},
	{"\"🏳️\u200d🌈\"", "🏳️\u200d🌈"},

	// Explicit tags.
	{
		"v: !!float '1.1'",
		map[string]any{"v": 1.1},
	},
	{
		"v: !!float 0",
		map[string]any{"v": float64(0)},
	},
	{
		"v: !!float -1",
		map[string]any{"v": float64(-1)},
	},
	{
		"v: !!null ''",
		map[string]any{"v": nil},
	},
	{
		"%TAG !y! tag:yaml.org,2002:\n---\nv: !y!int '1'",
		map[string]any{"v": 1},
	},
	// https://github.com/yaml/libyaml/pull/179.
	{
		"{\n  foo : !!str,\n  !!str : bar,\n}\n",
		map[string]any{"foo": "", "": "bar"},
	},

	// Non-specific tag (Issue #75)
	{
		"v: ! test",
		map[string]any{"v": "test"},
	},

	// Anchors and aliases.
	{
		"a: &x 1\nb: &y 2\nc: *x\nd: *y\n",
		&struct{ A, B, C, D int }{1, 2, 1, 2},
	},
	{
		"a: &a {c: 1}\nb: *a",
		&struct {
			A, B struct {
				C int
			}
		}{struct{ C int }{1}, struct{ C int }{1}},
	},
	{
		"a: &a [1, 2]\nb: *a",
		&struct{ B []int }{[]int{1, 2}},
	},
	// Unicode anchor.
	{
		"a: &🤡 [1, 2]\nb: *🤡",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		"a: &🏳️‍🌈 [1, 2]\nb: *🏳️‍🌈",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		"a: &👱🏻‍♀️ [1, 2]\nb: *👱🏻‍♀️",
		&struct{ B []int }{[]int{1, 2}},
	},
	// Test that YAML spec anchor names are accepted.
	//
	// See https://github.com/go-yaml/yaml/issues/920.
	// Testdata taken from https://github.com/go-yaml/yaml/pull/921.
	{
		// >= 0x21
		"a: &! [1, 2]\nb: *!",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		// <= 0x7E
		"a: &~ [1, 2]\nb: *~",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		// >= 0xA0 (Start of Basic Multilingual Plane)
		"a: &\u00A0 [1, 2]\nb: *\u00A0",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		// <= 0xD7FF (End of Basic Multilingual Plane)
		"a: &\uD7FF [1, 2]\nb: *\uD7FF",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		// >= 0xE000 (Start of Private Use area)
		"a: &\uE000 [1, 2]\nb: *\uE000",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		// <= 0xFFFD (End of allowed Private Use Area)
		"a: &\uFFFD [1, 2]\nb: *\uFFFD",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		// >= 0x010000 (Start of Supplementary Planes)
		"a: &\U00010000 [1, 2]\nb: *\U00010000",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		// >= 0x10FFFF (End of Supplementary Planes)
		"a: &\U0010FFFF [1, 2]\nb: *\U0010FFFF",
		&struct{ B []int }{[]int{1, 2}},
	},

	// Bug #1133337
	{
		"foo: ''",
		map[string]*string{"foo": new(string)},
	},
	{
		"foo: null",
		map[string]*string{"foo": nil},
	},
	{
		"foo: null",
		map[string]string{"foo": ""},
	},
	{
		"foo: null",
		map[string]any{"foo": nil},
	},

	// Support for ~
	{
		"foo: ~",
		map[string]*string{"foo": nil},
	},
	{
		"foo: ~",
		map[string]string{"foo": ""},
	},
	{
		"foo: ~",
		map[string]any{"foo": nil},
	},

	// Ignored field
	{
		"a: 1\nb: 2\n",
		&struct {
			A int
			B int "-"
		}{1, 0},
	},

	// Bug #1191981
	{
		"" +
			"%YAML 1.1\n" +
			"--- !!str\n" +
			`"Generic line break (no glyph)\n\` + "\n" +
			` Generic line break (glyphed)\n\` + "\n" +
			` Line separator\u2028\` + "\n" +
			` Paragraph separator\u2029"` + "\n",
		"" +
			"Generic line break (no glyph)\n" +
			"Generic line break (glyphed)\n" +
			"Line separator\u2028Paragraph separator\u2029",
	},

	// Struct inlining
	{
		"a: 1\nb: 2\nc: 3\n",
		&struct {
			A int
			C inlineB `yaml:",inline"`
		}{1, inlineB{2, inlineC{3}}},
	},

	// Struct inlining as a pointer.
	{
		"a: 1\nb: 2\nc: 3\n",
		&struct {
			A int
			C *inlineB `yaml:",inline"`
		}{1, &inlineB{2, inlineC{3}}},
	},
	{
		"a: 1\n",
		&struct {
			A int
			C *inlineB `yaml:",inline"`
		}{1, nil},
	},
	{
		"a: 1\nc: 3\nd: 4\n",
		&struct {
			A int
			C *inlineD `yaml:",inline"`
		}{1, &inlineD{&inlineC{3}, 4}},
	},

	// Map inlining
	{
		"a: 1\nb: 2\nc: 3\n",
		&struct {
			A int
			C map[string]int `yaml:",inline"`
		}{1, map[string]int{"b": 2, "c": 3}},
	},

	// bug 1243827
	{
		"a: -b_c",
		map[string]any{"a": "-b_c"},
	},
	{
		"a: +b_c",
		map[string]any{"a": "+b_c"},
	},
	{
		"a: 50cent_of_dollar",
		map[string]any{"a": "50cent_of_dollar"},
	},

	// issue #295 (allow scalars with colons in flow mappings and sequences)
	{
		"a: {b: https://github.com/go-yaml/yaml}",
		map[string]any{"a": map[string]any{
			"b": "https://github.com/go-yaml/yaml",
		}},
	},
	{
		"a: [https://github.com/go-yaml/yaml]",
		map[string]any{"a": []any{"https://github.com/go-yaml/yaml"}},
	},

	// Duration
	{
		"a: 3s",
		map[string]time.Duration{"a": 3 * time.Second},
	},

	// Issue #24.
	{
		"a: <foo>",
		map[string]string{"a": "<foo>"},
	},

	// Base 60 floats are obsolete and unsupported.
	{
		"a: 1:1\n",
		map[string]string{"a": "1:1"},
	},

	// Binary data.
	{
		"a: !!binary gIGC\n",
		map[string]string{"a": "\x80\x81\x82"},
	},
	{
		"a: !!binary |\n  " + strings.Repeat("kJCQ", 17) + "kJ\n  CQ\n",
		map[string]string{"a": strings.Repeat("\x90", 54)},
	},
	{
		"a: !!binary |\n  " + strings.Repeat("A", 70) + "\n  ==\n",
		map[string]string{"a": strings.Repeat("\x00", 52)},
	},

	// Issue #39.
	{
		"a:\n b:\n  c: d\n",
		map[string]struct{ B any }{"a": {map[string]any{"c": "d"}}},
	},

	// Custom map type.
	{
		"a: {b: c}",
		M{"a": M{"b": "c"}},
	},
	// Complex key in map.
	{
		"[0, 1]: foo\n[1, 0]: bar\n",
		map[[2]int]string{
			{0, 1}: "foo",
			{1, 0}: "bar",
		},
	},
	{
		"{a: A1, b: B1}: V1\n{b: B1}: V2\n",
		map[struct {
			A string `yaml:"a,omitempty"`
			B string `yaml:"b"`
		}]string{
			{"A1", "B1"}: "V1",
			{"", "B1"}:   "V2",
		},
	},
	// https://github.com/go-yaml/yaml/issues/890
	// https://github.com/go-yaml/yaml/pull/889
	{
		"?\n  K1\n: V1",
		map[string]string{
			"K1": "V1",
		},
	},
	{
		"?\n  a: A1\n  b: B1\n: V1",
		map[struct{ A, B string }]string{
			{"A1", "B1"}: "V1",
		},
	},
	{
		"?\n  - A1\n  - B1\n: V1",
		map[[2]string]string{
			{"A1", "B1"}: "V1",
		},
	},
	{
		"?\n  a: A1\n  b: B1\n: V1\n?\n  a: A2\n  b: B2\n: V2",
		map[struct{ A, B string }]string{
			{"A1", "B1"}: "V1",
			{"A2", "B2"}: "V2",
		},
	},
	{
		"?\n  - A1\n  - B1\n: V1\n?\n  - A2\n  - B2\n: V2",
		map[[2]string]string{
			{"A1", "B1"}: "V1",
			{"A2", "B2"}: "V2",
		},
	},

	// Support encoding.TextUnmarshaler.
	{
		"a: 1.2.3.4\n",
		map[string]textUnmarshaler{"a": {S: "1.2.3.4"}},
	},
	{
		"a: 2015-02-24T18:19:39Z\n",
		map[string]textUnmarshaler{"a": {"2015-02-24T18:19:39Z"}},
	},

	// Timestamps
	{
		// Date only.
		"a: 2015-01-01\n",
		map[string]time.Time{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},
	{
		// RFC3339
		"a: 2015-02-24T18:19:39.12Z\n",
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, .12e9, time.UTC)},
	},
	{
		// RFC3339 with short dates.
		"a: 2015-2-3T3:4:5Z",
		map[string]time.Time{"a": time.Date(2015, 2, 3, 3, 4, 5, 0, time.UTC)},
	},
	{
		// ISO8601 lower case t
		"a: 2015-02-24t18:19:39Z\n",
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, 0, time.UTC)},
	},
	{
		// space separate, no time zone
		"a: 2015-02-24 18:19:39\n",
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, 0, time.UTC)},
	},
	// Some cases not currently handled. Uncomment these when
	// the code is fixed.
	//	{
	//		// space separated with time zone
	//		"a: 2001-12-14 21:59:43.10 -5",
	//		map[string]interface{}{"a": time.Date(2001, 12, 14, 21, 59, 43, .1e9, time.UTC)},
	//	},
	//	{
	//		// arbitrary whitespace between fields
	//		"a: 2001-12-14 \t\t \t21:59:43.10 \t Z",
	//		map[string]interface{}{"a": time.Date(2001, 12, 14, 21, 59, 43, .1e9, time.UTC)},
	//	},
	{
		// explicit string tag
		"a: !!str 2015-01-01",
		map[string]any{"a": "2015-01-01"},
	},
	{
		// explicit timestamp tag on quoted string
		"a: !!timestamp \"2015-01-01\"",
		map[string]time.Time{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},
	{
		// explicit timestamp tag on unquoted string
		"a: !!timestamp 2015-01-01",
		map[string]time.Time{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},
	{
		// quoted string that's a valid timestamp
		"a: \"2015-01-01\"",
		map[string]any{"a": "2015-01-01"},
	},
	{
		// explicit timestamp tag into interface.
		"a: !!timestamp \"2015-01-01\"",
		map[string]any{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},
	{
		// implicit timestamp tag into interface.
		"a: 2015-01-01",
		map[string]any{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},

	// Encode empty lists as zero-length slices.
	{
		"a: []",
		&struct{ A []int }{[]int{}},
	},

	// UTF-16-LE
	{
		"\xff\xfe\xf1\x00o\x00\xf1\x00o\x00:\x00 \x00v\x00e\x00r\x00y\x00 \x00y\x00e\x00s\x00\n\x00",
		M{"ñoño": "very yes"},
	},
	// UTF-16-LE with surrogate.
	{
		"\xff\xfe\xf1\x00o\x00\xf1\x00o\x00:\x00 \x00v\x00e\x00r\x00y\x00 \x00y\x00e\x00s\x00 \x00=\xd8\xd4\xdf\n\x00",
		M{"ñoño": "very yes 🟔"},
	},

	// UTF-16-BE
	{
		"\xfe\xff\x00\xf1\x00o\x00\xf1\x00o\x00:\x00 \x00v\x00e\x00r\x00y\x00 \x00y\x00e\x00s\x00\n",
		M{"ñoño": "very yes"},
	},
	// UTF-16-BE with surrogate.
	{
		"\xfe\xff\x00\xf1\x00o\x00\xf1\x00o\x00:\x00 \x00v\x00e\x00r\x00y\x00 \x00y\x00e\x00s\x00 \xd8=\xdf\xd4\x00\n",
		M{"ñoño": "very yes 🟔"},
	},

	// This *is* in fact a float number, per the spec. #171 was a mistake.
	{
		"a: 123456e1\n",
		M{"a": 123456e1},
	},
	{
		"a: 123456E1\n",
		M{"a": 123456e1},
	},
	// yaml-test-suite 3GZX: Spec Example 7.1. Alias Nodes
	{
		"First occurrence: &anchor Foo\nSecond occurrence: *anchor\nOverride anchor: &anchor Bar\nReuse anchor: *anchor\n",
		map[string]any{
			"First occurrence":  "Foo",
			"Second occurrence": "Foo",
			"Override anchor":   "Bar",
			"Reuse anchor":      "Bar",
		},
	},
	// Single document with garbage following it.
	{
		"---\nhello\n...\n}not yaml",
		"hello",
	},

	// Comment scan exhausting the input buffer (issue #469).
	{
		"true\n#" + strings.Repeat(" ", 512*3),
		"true",
	},
	{
		"true #" + strings.Repeat(" ", 512*3),
		"true",
	},

	// CRLF
	{
		"a: b\r\nc:\r\n- d\r\n- e\r\n",
		map[string]any{
			"a": "b",
			"c": []any{"d", "e"},
		},
	},
}

type M map[string]any

type inlineB struct {
	B       int
	inlineC `yaml:",inline"`
}

type inlineC struct {
	C int
}

type inlineD struct {
	C *inlineC `yaml:",inline"`
	D int
}

func TestUnmarshal(t *testing.T) {
	for i, item := range unmarshalTests {
		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %q", item.data)
			a := require.New(t)

			typ := reflect.ValueOf(item.value).Type()
			value := reflect.New(typ)
			err := yaml.Unmarshal([]byte(item.data), value.Interface())
			if _, ok := err.(*yaml.TypeError); !ok {
				a.NoError(err)
			}
			a.Equal(item.value, value.Elem().Interface())
		})
	}
}

func TestUnmarshalFullTimestamp(t *testing.T) {
	a := require.New(t)

	// Full timestamp in same format as encoded. This is confirmed to be
	// properly decoded by Python as a timestamp as well.
	str := "2015-02-24T18:19:39.123456789-03:00"
	var i any
	err := yaml.Unmarshal([]byte(str), &i)
	a.NoError(err)
	a.IsType(time.Time{}, i)
	input := i.(time.Time)

	expected := time.Date(
		2015, 2, 24, 18, 19, 39, 123456789,
		input.Location(),
	)
	a.Equal(expected, input)

	expected = time.Date(
		2015, 2, 24, 21, 19, 39, 123456789,
		time.UTC,
	)
	a.Equal(expected, input.In(time.UTC))
}

func TestDecoderSingleDocument(t *testing.T) {
	// Test that Decoder.Decode works as expected on
	// all the unmarshal tests.
	for i, item := range unmarshalTests {
		if item.data == "" {
			// Behavior differs when there's no YAML.
			continue
		}

		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %q", item.data)

			testReader(t, func(wrap func(io.Reader) io.Reader) func(t *testing.T) {
				return func(t *testing.T) {
					a := require.New(t)

					typ := reflect.ValueOf(item.value).Type()
					value := reflect.New(typ)

					r := wrap(strings.NewReader(item.data))
					err := yaml.NewDecoder(r).Decode(value.Interface())
					if _, ok := err.(*yaml.TypeError); !ok {
						a.NoError(err)
					}
					a.Equal(item.value, value.Elem().Interface())
				}
			})
		})
	}
}

var decoderTests = []struct {
	data   string
	values []any
}{{
	"",
	nil,
}, {
	"a: b",
	[]any{
		map[string]any{"a": "b"},
	},
}, {
	"---\na: b\n...\n",
	[]any{
		map[string]any{"a": "b"},
	},
}, {
	"---\n'hello'\n...\n---\ngoodbye\n...\n",
	[]any{
		"hello",
		"goodbye",
	},
}}

func TestDecoder(t *testing.T) {
	for i, item := range decoderTests {
		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %q", item.data)
			testReader(t, func(wrap func(io.Reader) io.Reader) func(t *testing.T) {
				return func(t *testing.T) {
					a := require.New(t)
					var (
						values []any
						r      = wrap(strings.NewReader(item.data))
						dec    = yaml.NewDecoder(r)
					)
					for {
						var value any
						err := dec.Decode(&value)
						if err == io.EOF {
							break
						}
						a.NoError(err)
						values = append(values, value)
					}
					a.Equal(item.values, values)
				}
			})
		})
	}
}

func TestDecoderReadError(t *testing.T) {
	var testError = errors.New("some read error")
	err := yaml.NewDecoder(iotest.ErrReader(testError)).Decode(&struct{}{})
	require.ErrorIs(t, err, testError)
}

func TestUnmarshalNaN(t *testing.T) {
	a := require.New(t)

	value := map[string]any{}
	err := yaml.Unmarshal([]byte("notanum: .NaN"), &value)
	a.NoError(err)
	a.True(math.IsNaN(value["notanum"].(float64)))
}

func TestUnmarshalDurationInt(t *testing.T) {
	a := require.New(t)

	// Don't accept plain ints as durations as it's unclear (issue #200).
	var d time.Duration
	err := yaml.Unmarshal([]byte("123"), &d)
	a.Error(err)
	a.Regexp("(?s).* line 1: cannot unmarshal !!int `123` into time.Duration", err.Error())
}

func TestUnmarshalArray(t *testing.T) {
	tests := []struct {
		data  string
		value any
		err   string
	}{
		{"- 1\n- 2\n- 3", [3]int{1, 2, 3}, ""},
		{"[]", [0]int{}, ""},
		{"b: 0", struct {
			A [3]int `yaml:"a"`
			B int
		}{}, ""},

		{"- 1\n- 2\n- 3", [4]int{}, "yaml: line 1: invalid array: want 4 elements but got 3"},
		{"- 1\n- 2\n- 3", [1]int{}, "yaml: line 1: invalid array: want 1 elements but got 3"},
		{"- 1\n- 2\n- 3", [0]int{}, "yaml: line 1: invalid array: want 0 elements but got 3"},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			a := require.New(t)

			typ := reflect.ValueOf(tt.value).Type()
			value := reflect.New(typ)
			err := yaml.Unmarshal([]byte(tt.data), value.Interface())
			if tt.err != "" {
				a.Error(err)
				a.Regexp(tt.err, err.Error())
				return
			}
			a.NoError(err)
			a.Equal(tt.value, value.Elem().Interface())
		})
	}
}

func TestUnmarshalNodeMap(t *testing.T) {
	t.Run("Inline", func(t *testing.T) {
		t.Run("Value", func(t *testing.T) {
			a := require.New(t)

			var val struct {
				A      int
				Inline map[string]yaml.Node `yaml:",inline"`
			}
			a.NoError(yaml.Unmarshal([]byte("a: 1\nb: 2\nc: 3"), &val))

			a.Equal(1, val.A)
			inline := val.Inline
			a.Equal("2", inline["b"].Value)
			a.Equal("3", inline["c"].Value)
		})
		t.Run("Pointer", func(t *testing.T) {
			a := require.New(t)

			var val struct {
				A      int
				Inline map[string]*yaml.Node `yaml:",inline"`
			}
			a.NoError(yaml.Unmarshal([]byte("a: 1\nb: 2\nc: 3"), &val))

			a.Equal(1, val.A)
			inline := val.Inline
			a.Equal("2", inline["b"].Value)
			a.Equal("3", inline["c"].Value)
		})
	})
	t.Run("Issue769", func(t *testing.T) {
		// https://github.com/go-yaml/yaml/issues/769
		a := require.New(t)

		var val struct {
			Foo  map[string]*yaml.Node `yaml:"foo"`
			Bang int
		}
		a.NoError(yaml.Unmarshal([]byte(`
foo:
  bar: !hello
    - !a 11
    - !b 2
    - !c 3
  buz: !hi
    - 4
    - 5
bang: 12
`), &val))

		a.Equal(12, val.Bang)
		a.Len(val.Foo, 2)

		type nodeExpect struct {
			tag   string
			value string
		}
		checkContent := func(n *yaml.Node, expect []nodeExpect) {
			a.Equal(yaml.SequenceNode, n.Kind)
			c := n.Content
			a.Len(c, len(expect))

			for i, expect := range expect {
				elem := c[i]
				a.Equal(yaml.ScalarNode, elem.Kind)
				a.Equal(expect.tag, elem.Tag)
				a.Equal(expect.value, elem.Value)
			}
		}
		{
			bar, ok := val.Foo["bar"]
			a.True(ok)
			a.Equal("!hello", bar.Tag)
			checkContent(bar, []nodeExpect{
				{"!a", "11"},
				{"!b", "2"},
				{"!c", "3"},
			})
		}

		{
			buz, ok := val.Foo["buz"]
			a.True(ok)
			checkContent(buz, []nodeExpect{
				{"!!int", "4"},
				{"!!int", "5"},
			})
		}
	})
}

var unmarshalErrorTests = []struct {
	data, error string
}{
	{"v: !!float 'error'", "yaml: cannot decode !!str `error` as a !!float"},
	{"v: [A,", "yaml: line 1: did not find expected node content"},
	{"v:\n- [A,", "yaml: line 2: did not find expected node content"},
	{"a:\n- b: *,", "yaml: line 2:5: did not find expected alphabetic or numeric character"},
	{"a: *b\n", "yaml: line 1: unknown anchor \"b\" referenced"},
	{"a: &a\n  b: *a\n", "yaml: line 2: anchor \"a\" value contains itself"},
	{"b: *a\na: &a {c: 1}", `yaml: line 1: unknown anchor "a" referenced`},
	{"value: -", "yaml: offset 7: block sequence entries are not allowed in this context"},
	{"a: !!binary ==", "yaml: line 1: decode !!binary: illegal base64 data at input byte 0"},
	{"{[.]}", `yaml: line 1: invalid map key: \[\]interface \{\}\{"\."\}`},
	{"{{.}}", `yaml: line 1: invalid map key: map\[string]interface \{\}\{".":interface \{\}\(nil\)\}`},
	{"%TAG !%79! tag:yaml.org,2002:\n---\nv: !%79!int '1'", "yaml: offset 6: did not find expected whitespace"},
	{"a:\n  1:\nb\n  2:", ".*could not find expected ':'"},
	{"a: 1\nb: 2\nc 2\nd: 3\n", "^yaml: line 3: could not find expected ':'$"},
	{"#\n-\n{", "yaml: line 3: could not find expected ':'"},             // Issue #665
	{"0: [:!00 \xef", "yaml: offset 9: incomplete UTF-8 octet sequence"}, // Issue #666
	{
		"a: &a [00,00,00,00,00,00,00,00,00]\n" +
			"b: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a]\n" +
			"c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b]\n" +
			"d: &d [*c,*c,*c,*c,*c,*c,*c,*c,*c]\n" +
			"e: &e [*d,*d,*d,*d,*d,*d,*d,*d,*d]\n" +
			"f: &f [*e,*e,*e,*e,*e,*e,*e,*e,*e]\n" +
			"g: &g [*f,*f,*f,*f,*f,*f,*f,*f,*f]\n" +
			"h: &h [*g,*g,*g,*g,*g,*g,*g,*g,*g]\n" +
			"i: &i [*h,*h,*h,*h,*h,*h,*h,*h,*h]\n",
		"yaml: line 2: document contains excessive aliasing",
	},

	// Duplicate keys.
	{"a: 0\na: 1", `yaml: line 2: mapping key "a" already defined at line 1`},
	{"10: 0\n10: 1", `yaml: line 2: mapping key "10" already defined at line 1`},
	{"true: 0\ntrue: 1", `yaml: line 2: mapping key "true" already defined at line 1`},
	{"false: 0\nfalse: 1", `yaml: line 2: mapping key "false" already defined at line 1`},
	{"[]: 0\n[]: 1", `yaml: line 2: mapping key already defined at line 1`},
	{"{}: 0\n{}: 1", `yaml: line 2: mapping key already defined at line 1`},
	{"{foo: 0, bar: 0}: 0\n{bar: 0, foo: 0}: 1", `yaml: line 2: mapping key already defined at line 1`},

	// https://github.com/yaml/libyaml/issues/68
	{"double: \"quoted \\' scalar\"", "yaml: offset 16: found unknown escape character"},

	// Invalid surrogate pair.
	{`"\ud800\ud800"`, "yaml: offset 9: found invalid Unicode character escape code"},
	// Invalid Unicode.
	{`"\U00FFFFFF"`, "yaml: offset 3: found invalid Unicode character escape code"},

	// Invalid indentation (unexpected tab).
	{"- >\n \t\n\tdetected\n", "yaml: line 3: found a tab character where an indentation space is expected"},

	// https://github.com/go-faster/yaml/issues/20
	{"0:\n00:\n 000\n<<:\n  {}:", `yaml: line 5: invalid map key: map\[string\]interface \{\}\{\}`},
	{"0:\n00:\n 000\n<<:\n  []:", `yaml: line 5: invalid map key: \[\]interface \{\}\{\}`},
	{"{}:", `yaml: line 1: invalid map key: map\[string\]interface \{\}\{\}`},
	{"[]:", `yaml: line 1: invalid map key: \[\]interface \{\}\{\}`},

	// Invalid anchor/alias name.
	{"a: &", "yaml: offset 4: did not find expected alphabetic or numeric character"},
	{"a: *", "yaml: offset 4: did not find expected alphabetic or numeric character"},
	{"a: & foo\n", "yaml: offset 4: did not find expected alphabetic or numeric character"},
	{"a: &, foo\n", "yaml: offset 4: did not find expected alphabetic or numeric character"},
	{"a: foo\nb: *\n", "yaml: line 2:3: did not find expected alphabetic or numeric character"},
	{"a: foo\nb: *,\n", "yaml: line 2:3: did not find expected alphabetic or numeric character"},

	// From https://github.com/go-yaml/yaml/pull/921.
	{"a:\n- b: *,", `yaml: line 2:5: did not find expected alphabetic or numeric character`},
	{"a:\n- b: *a{", `yaml: line 2:5: did not find expected alphabetic or numeric character`},
	{"a:\n- b: *a\u0019", `yaml: offset 10: control characters are not allowed`},
	{"a:\n- b: *a\u0020", `yaml: line 2: unknown anchor "a" referenced`},
	{"a:\n- b: *a\u007F", `yaml: offset 10: control characters are not allowed`},
	{"a:\n- b: *a\u0099", `yaml: offset 10: control characters are not allowed`},
	{"a:\n- b: *a\uFFFE", `yaml: offset 10: control characters are not allowed`},
	{"a:\n- b: *a\uFFFF", `yaml: offset 10: control characters are not allowed`},
}

func TestUnmarshalErrors(t *testing.T) {
	for i, item := range unmarshalErrorTests {
		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %q", item.data)
			a := require.New(t)

			var value any
			err := yaml.Unmarshal([]byte(item.data), &value)
			a.Error(err)
			a.Regexp(item.error, err.Error())
		})
	}
}

func TestDecoderErrors(t *testing.T) {
	for i, item := range unmarshalErrorTests {
		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %q", item.data)
			a := require.New(t)

			var value any
			err := yaml.NewDecoder(strings.NewReader(item.data)).Decode(&value)
			a.Error(err)
			a.Regexp(item.error, err.Error())
		})
	}
}

var unmarshalerTests = []struct {
	data, tag string
	value     any
}{
	{"_: {hi: there}", "!!map", map[string]any{"hi": "there"}},
	{"_: [1,A]", "!!seq", []any{1, "A"}},
	{"_: 10", "!!int", 10},
	{"_: null", "!!null", nil},
	{`_: BAR!`, "!!str", "BAR!"},
	{`_: "BAR!"`, "!!str", "BAR!"},
	{"_: !!foo 'BAR!'", "!!foo", "BAR!"},
	{`_: ""`, "!!str", ""},
}

var unmarshalerResult = map[int]error{}

type unmarshalerType struct {
	value any
}

func (o *unmarshalerType) UnmarshalYAML(value *yaml.Node) error {
	if err := value.Decode(&o.value); err != nil {
		return err
	}
	if i, ok := o.value.(int); ok {
		if result, ok := unmarshalerResult[i]; ok {
			return result
		}
	}
	return nil
}

type unmarshalerPointer struct {
	Field *unmarshalerType "_"
}

type unmarshalerInlined struct {
	Field   *unmarshalerType "_"
	Inlined unmarshalerType  `yaml:",inline"`
}

type unmarshalerInlinedTwice struct {
	InlinedTwice unmarshalerInlined `yaml:",inline"`
}

type obsoleteUnmarshalerType struct {
	value any
}

func (o *obsoleteUnmarshalerType) UnmarshalYAML(unmarshal func(v any) error) error {
	if err := unmarshal(&o.value); err != nil {
		return err
	}
	if i, ok := o.value.(int); ok {
		if result, ok := unmarshalerResult[i]; ok {
			return result
		}
	}
	return nil
}

type obsoleteUnmarshalerPointer struct {
	Field *obsoleteUnmarshalerType "_"
}

type obsoleteUnmarshalerValue struct {
	Field obsoleteUnmarshalerType "_"
}

func TestUnmarshalerPointerField(t *testing.T) {
	t.Run("Unmarshaler", func(t *testing.T) {
		for i, item := range unmarshalerTests {
			item := item
			t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
				t.Logf("Input: %q", item.data)
				a := require.New(t)

				obj := &unmarshalerPointer{}
				err := yaml.Unmarshal([]byte(item.data), obj)
				a.NoError(err)

				if item.value == nil {
					a.Nil(obj.Field)
				} else {
					a.NotNil(obj.Field)
					a.Equal(item.value, obj.Field.value)
				}
			})
		}
	})

	t.Run("ObsoleteUnmarshaler", func(t *testing.T) {
		for i, item := range unmarshalerTests {
			item := item
			t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
				t.Logf("Input: %q", item.data)
				a := require.New(t)

				obj := &obsoleteUnmarshalerPointer{}
				err := yaml.Unmarshal([]byte(item.data), obj)
				a.NoError(err)

				if item.value == nil {
					a.Nil(obj.Field)
				} else {
					a.NotNil(obj.Field)
					a.Equal(item.value, obj.Field.value)
				}
			})
		}
	})
}

func TestUnmarshalerValueField(t *testing.T) {
	t.Run("Unmarshaler", func(t *testing.T) {
		for i, item := range unmarshalerTests {
			item := item
			t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
				t.Logf("Input: %q", item.data)
				a := require.New(t)

				obj := &obsoleteUnmarshalerValue{}
				err := yaml.Unmarshal([]byte(item.data), obj)
				a.NoError(err)
				a.Equal(item.value, obj.Field.value)
			})
		}
	})
}

func TestUnmarshalerInlinedField(t *testing.T) {
	t.Run("Inlined", func(t *testing.T) {
		a := require.New(t)

		obj := &unmarshalerInlined{}
		err := yaml.Unmarshal([]byte("_: a\ninlined: b\n"), obj)
		a.NoError(err)
		a.Equal(&unmarshalerType{"a"}, obj.Field)
		a.Equal(unmarshalerType{map[string]any{"_": "a", "inlined": "b"}}, obj.Inlined)
	})

	t.Run("InlinedTwice", func(t *testing.T) {
		a := require.New(t)

		obj := &unmarshalerInlinedTwice{}
		err := yaml.Unmarshal([]byte("_: a\ninlined: b\n"), obj)
		a.NoError(err)
		a.Equal(&unmarshalerType{"a"}, obj.InlinedTwice.Field)
		a.Equal(unmarshalerType{map[string]any{"_": "a", "inlined": "b"}}, obj.InlinedTwice.Inlined)
	})
}

func TestUnmarshalerWholeDocument(t *testing.T) {
	a := require.New(t)

	obj := &obsoleteUnmarshalerType{}
	err := yaml.Unmarshal([]byte(unmarshalerTests[0].data), obj)
	a.NoError(err)

	value, ok := obj.value.(map[string]any)
	a.True(ok)
	a.Equal(unmarshalerTests[0].value, value["_"])
}

type recursiveUnmarshalerProperty struct {
	Name  string
	Value *recursiveUnmarshalerSchema
}

type recursiveUnmarshalerProperties []recursiveUnmarshalerProperty

type recursiveUnmarshalerSchema struct {
	Properties recursiveUnmarshalerProperties `yaml:"properties"`
}

func (p *recursiveUnmarshalerProperties) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.New("expected mapping node")
	}
	for i := 0; i < len(node.Content); i += 2 {
		var (
			key    = node.Content[i]
			value  = node.Content[i+1]
			schema *recursiveUnmarshalerSchema
		)
		if err := value.Decode(&schema); err != nil {
			return err
		}
		*p = append(*p, recursiveUnmarshalerProperty{
			Name:  key.Value,
			Value: schema,
		})
	}
	return nil
}

func TestUnmarshalerRecursiveAlias(t *testing.T) {
	// Check that recursive aliases are handled correctly
	// even if type defines custom unmarshaler.
	const input = `a:
  properties:
    foo: &foo
      properties:
        bar: *foo`

	var spec map[string]*recursiveUnmarshalerSchema
	err := yaml.Unmarshal([]byte(input), &spec)
	require.EqualError(t, err, "yaml: line 5: anchor \"foo\" value contains itself")
}

var errFailing = errors.New("errFailing")

type failingUnmarshaler struct{}

func (ft *failingUnmarshaler) UnmarshalYAML(node *yaml.Node) error {
	return errFailing
}

func TestUnmarshalerError(t *testing.T) {
	err := yaml.Unmarshal([]byte("a: b"), &failingUnmarshaler{})
	require.ErrorIs(t, err, errFailing)
}

type obsoleteFailingUnmarshaler struct{}

func (ft *obsoleteFailingUnmarshaler) UnmarshalYAML(unmarshal func(any) error) error {
	return errFailing
}

func TestObsoleteUnmarshalerError(t *testing.T) {
	err := yaml.Unmarshal([]byte("a: b"), &obsoleteFailingUnmarshaler{})
	require.ErrorIs(t, err, errFailing)
}

type sliceUnmarshaler []int

func (su *sliceUnmarshaler) UnmarshalYAML(node *yaml.Node) error {
	var slice []int
	err := node.Decode(&slice)
	if err == nil {
		*su = slice
		return nil
	}

	var intVal int
	err = node.Decode(&intVal)
	if err == nil {
		*su = []int{intVal}
		return nil
	}

	return err
}

func TestUnmarshalerRetry(t *testing.T) {
	a := require.New(t)

	var su sliceUnmarshaler
	err := yaml.Unmarshal([]byte("[1, 2, 3]"), &su)
	a.NoError(err)
	a.Equal(sliceUnmarshaler([]int{1, 2, 3}), su)

	err = yaml.Unmarshal([]byte("1"), &su)
	a.NoError(err)
	a.Equal(sliceUnmarshaler([]int{1}), su)
}

type obsoleteSliceUnmarshaler []int

func (su *obsoleteSliceUnmarshaler) UnmarshalYAML(unmarshal func(any) error) error {
	var slice []int
	err := unmarshal(&slice)
	if err == nil {
		*su = slice
		return nil
	}

	var intVal int
	err = unmarshal(&intVal)
	if err == nil {
		*su = []int{intVal}
		return nil
	}

	return err
}

func TestObsoleteUnmarshalerRetry(t *testing.T) {
	a := require.New(t)

	var su obsoleteSliceUnmarshaler
	err := yaml.Unmarshal([]byte("[1, 2, 3]"), &su)
	a.NoError(err)
	a.Equal(obsoleteSliceUnmarshaler([]int{1, 2, 3}), su)

	err = yaml.Unmarshal([]byte("1"), &su)
	a.NoError(err)
	a.Equal(obsoleteSliceUnmarshaler([]int{1}), su)
}

// From http://yaml.org/type/merge.html
var mergeTests = `
anchors:
  list:
    - &CENTER { "x": 1, "y": 2 }
    - &LEFT   { "x": 0, "y": 2 }
    - &BIG    { "r": 10 }
    - &SMALL  { "r": 1 }

# All the following maps are equal:

plain:
  # Explicit keys
  "x": 1
  "y": 2
  "r": 10
  label: center/big

mergeOne:
  # Merge one map
  << : *CENTER
  "r": 10
  label: center/big

mergeMultiple:
  # Merge multiple maps
  << : [ *CENTER, *BIG ]
  label: center/big

override:
  # Override
  << : [ *BIG, *LEFT, *SMALL ]
  "x": 1
  label: center/big

shortTag:
  # Explicit short merge tag
  !!merge "<<" : [ *CENTER, *BIG ]
  label: center/big

longTag:
  # Explicit merge long tag
  !<tag:yaml.org,2002:merge> "<<" : [ *CENTER, *BIG ]
  label: center/big

inlineMap:
  # Inlined map
  << : {"x": 1, "y": 2, "r": 10}
  label: center/big

inlineSequenceMap:
  # Inlined map in sequence
  << : [ *CENTER, {"r": 10} ]
  label: center/big
`

func TestMerge(t *testing.T) {
	a := require.New(t)

	want := map[string]any{
		"x":     1,
		"y":     2,
		"r":     10,
		"label": "center/big",
	}

	wantStringMap := make(map[string]any)
	for k, v := range want {
		wantStringMap[fmt.Sprintf("%v", k)] = v
	}

	var m map[any]any
	err := yaml.Unmarshal([]byte(mergeTests), &m)
	a.NoError(err)

	for name, test := range m {
		if name == "anchors" {
			continue
		}
		if name == "plain" {
			a.Equal(wantStringMap, test)
			continue
		}
		a.Equal(want, test)
	}
}

func TestMergeStruct(t *testing.T) {
	a := require.New(t)

	type Data struct {
		X, Y, R int
		Label   string
	}
	want := Data{1, 2, 10, "center/big"}

	var m map[string]Data
	err := yaml.Unmarshal([]byte(mergeTests), &m)
	a.NoError(err)

	for name, test := range m {
		if name == "anchors" {
			continue
		}
		a.Equal(want, test)
	}
}

var mergeTestsNested = `
mergeouter1: &mergeouter1
    d: 40
    e: 50

mergeouter2: &mergeouter2
    e: 5
    f: 6
    g: 70

mergeinner1: &mergeinner1
    <<: *mergeouter1
    inner:
        a: 1
        b: 2

mergeinner2: &mergeinner2
    <<: *mergeouter2
    inner:
        a: -1
        b: -2

outer:
    <<: [*mergeinner1, *mergeinner2]
    f: 60
    inner:
        a: 10
`

func TestMergeNestedStruct(t *testing.T) {
	// Issue #818: Merging used to just unmarshal twice on the target
	// value, which worked for maps as these were replaced by the new map,
	// but not on struct values as these are preserved. This resulted in
	// the nested data from the merged map to be mixed up with the data
	// from the map being merged into.
	//
	// This test also prevents two potential bugs from showing up:
	//
	// 1) A simple implementation might just zero out the nested value
	//    before unmarshaling the second time, but this would clobber previous
	//    data that is usually respected ({C: 30} below).
	//
	// 2) A simple implementation might attempt to handle the key skipping
	//    directly by iterating over the merging map without recursion, but
	//    there are more complex cases that require recursion.
	//
	// Quick summary of the fields:
	//
	// - A must come from outer and not overridden
	// - B must not be set as its in the ignored merge
	// - C should still be set as it's preset in the value
	// - D should be set from the recursive merge
	// - E should be set from the first recursive merge, ignored on the second
	// - F should be set in the inlined map from outer, ignored later
	// - G should be set in the inlined map from the second recursive merge
	//
	a := require.New(t)

	type Inner struct {
		A, B, C int
	}
	type Outer struct {
		D, E   int
		Inner  Inner
		Inline map[string]int `yaml:",inline"`
	}
	type Data struct {
		Outer Outer
	}

	test := Data{Outer{0, 0, Inner{C: 30}, nil}}
	want := Data{Outer{40, 50, Inner{A: 10, C: 30}, map[string]int{"f": 60, "g": 70}}}

	err := yaml.Unmarshal([]byte(mergeTestsNested), &test)
	a.NoError(err)
	a.Equal(want, test)

	// Repeat test with a map.

	var testm map[string]any
	wantm := map[string]any{
		"f": 60,
		"inner": map[string]any{
			"a": 10,
		},
		"d": 40,
		"e": 50,
		"g": 70,
	}
	err = yaml.Unmarshal([]byte(mergeTestsNested), &testm)
	a.NoError(err)
	a.Equal(wantm, testm["outer"])
}

var unmarshalNullTests = []struct {
	input              string
	pristine, expected func() any
}{{
	"null",
	func() any { var v any; v = "v"; return &v },
	func() any { var v any; v = nil; return &v },
}, {
	"null",
	func() any { s := "s"; return &s },
	func() any { s := "s"; return &s },
}, {
	"null",
	func() any { s := "s"; sptr := &s; return &sptr },
	func() any { var sptr *string; return &sptr },
}, {
	"null",
	func() any { i := 1; return &i },
	func() any { i := 1; return &i },
}, {
	"null",
	func() any { i := 1; iptr := &i; return &iptr },
	func() any { var iptr *int; return &iptr },
}, {
	"null",
	func() any { m := map[string]int{"s": 1}; return &m },
	func() any { var m map[string]int; return &m },
}, {
	"null",
	func() any { m := map[string]int{"s": 1}; return m },
	func() any { m := map[string]int{"s": 1}; return m },
}, {
	"s2: null\ns3: null",
	func() any { m := map[string]int{"s1": 1, "s2": 2}; return m },
	func() any { m := map[string]int{"s1": 1, "s2": 2, "s3": 0}; return m },
}, {
	"s2: null\ns3: null",
	func() any { m := map[string]any{"s1": 1, "s2": 2}; return m },
	func() any { m := map[string]any{"s1": 1, "s2": nil, "s3": nil}; return m },
}}

func TestUnmarshalNull(t *testing.T) {
	for i, test := range unmarshalNullTests {
		test := test
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %q", test.input)
			a := require.New(t)

			pristine := test.pristine()
			expected := test.expected()
			err := yaml.Unmarshal([]byte(test.input), pristine)
			a.NoError(err)
			a.Equal(expected, pristine)
		})
	}
}

func TestUnmarshalPreservesData(t *testing.T) {
	a := require.New(t)

	type typ struct {
		A, B int
		C    int `yaml:"-"`
	}
	var v typ
	v.A = 42
	v.C = 88
	err := yaml.Unmarshal([]byte("---"), &v)
	a.NoError(err)
	a.Equal(typ{42, 0, 88}, v)

	err = yaml.Unmarshal([]byte("b: 21\nc: 99"), &v)
	a.NoError(err)
	a.Equal(typ{42, 21, 88}, v)
}

func TestUnmarshalSliceOnPreset(t *testing.T) {
	a := require.New(t)

	// Issue #48.
	v := struct{ A []int }{[]int{1}}
	err := yaml.Unmarshal([]byte("a: [2]"), &v)
	a.NoError(err)
	a.Equal([]int{2}, v.A)
}

var unmarshalStrictTests = []struct {
	known  bool
	unique bool
	data   string
	value  any
	error  string
}{{
	known: true,
	data:  "a: 1\nc: 2\n",
	value: struct{ A, B int }{A: 1},
	error: `yaml: unmarshal errors:\n  yaml: line 2: field "c" not found in type struct { A int; B int }`,
}, {
	unique: true,
	data:   "a: 1\nb: 2\na: 3\n",
	value:  struct{ A, B int }{A: 3, B: 2},
	error:  `yaml: unmarshal errors:\n  yaml: line 3: mapping key "a" already defined at line 1`,
}, {
	unique: true,
	data:   "c: 3\na: 1\nb: 2\nc: 4\n",
	value: struct {
		A       int
		inlineB `yaml:",inline"`
	}{
		A: 1,
		inlineB: inlineB{
			B: 2,
			inlineC: inlineC{
				C: 4,
			},
		},
	},
	error: `yaml: unmarshal errors:\n  yaml: line 4: mapping key "c" already defined at line 1`,
}, {
	unique: true,
	data:   "c: 0\na: 1\nb: 2\nc: 1\n",
	value: struct {
		A       int
		inlineB `yaml:",inline"`
	}{
		A: 1,
		inlineB: inlineB{
			B: 2,
			inlineC: inlineC{
				C: 1,
			},
		},
	},
	error: `yaml: unmarshal errors:\n  yaml: line 4: mapping key "c" already defined at line 1`,
}, {
	unique: true,
	data:   "c: 1\na: 1\nb: 2\nc: 3\n",
	value: struct {
		A int
		M map[string]any `yaml:",inline"`
	}{
		A: 1,
		M: map[string]any{
			"b": 2,
			"c": 3,
		},
	},
	error: `yaml: unmarshal errors:\n  yaml: line 4: mapping key "c" already defined at line 1`,
}, {
	unique: true,
	data:   "a: 1\n9: 2\nnull: 3\n9: 4",
	value: map[any]any{
		"a": 1,
		nil: 3,
		9:   4,
	},
	error: `yaml: unmarshal errors:\n  yaml: line 4: mapping key "9" already defined at line 2`,
}}

func TestUnmarshalKnownFields(t *testing.T) {
	for i, item := range unmarshalStrictTests {
		item := item
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Logf("Input: %q", item.data)
			a := require.New(t)

			// First test that normal Unmarshal unmarshals to the expected value.
			if !item.unique {
				t := reflect.ValueOf(item.value).Type()
				value := reflect.New(t)
				err := yaml.Unmarshal([]byte(item.data), value.Interface())
				a.NoError(err)
				a.Equal(item.value, value.Elem().Interface())
			}

			// Then test that it fails on the same thing with KnownFields on.
			typ := reflect.ValueOf(item.value).Type()
			value := reflect.New(typ)
			dec := yaml.NewDecoder(bytes.NewBuffer([]byte(item.data)))
			dec.KnownFields(item.known)
			err := dec.Decode(value.Interface())
			a.Error(err)
			a.Regexp(item.error, err.Error())
		})
	}
}

type textUnmarshaler struct {
	S string
}

func (t *textUnmarshaler) UnmarshalText(s []byte) error {
	t.S = string(s)
	return nil
}
