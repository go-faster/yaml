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

package yaml

import (
	"encoding"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"reflect"
	"time"

	"go.uber.org/multierr"

	"github.com/go-faster/errors"
)

// ----------------------------------------------------------------------------
// Parser, produces a node tree out of a libyaml event stream.

type parser struct {
	parser        yaml_parser_t
	event         yaml_event_t
	doc           *Node
	anchors       map[string]*Node
	parentAnchors map[string]struct{}
	doneInit      bool
	textless      bool
}

func newParser(b []byte) *parser {
	p := getParser()
	if len(b) == 0 {
		b = []byte{'\n'}
	}
	yaml_parser_set_input_string(&p.parser, b)
	return p
}

func newParserFromReader(r io.Reader) *parser {
	p := getParser()
	yaml_parser_set_input_reader(&p.parser, r)
	return p
}

func (p *parser) init() {
	if p.doneInit {
		return
	}
	p.anchors = make(map[string]*Node)
	p.parentAnchors = make(map[string]struct{})
	p.expect(yaml_STREAM_START_EVENT)
	p.doneInit = true
}

func (p *parser) destroy() {
	putParser(p)
}

// expect consumes an event from the event stream and
// checks that it's of the expected type.
func (p *parser) expect(e yaml_event_type_t) {
	if p.event.typ == yaml_NO_EVENT {
		if !yaml_parser_parse(&p.parser, &p.event) {
			p.fail()
		}
	}
	if p.event.typ == yaml_STREAM_END_EVENT {
		p.parser.problem = "attempted to go past the end of stream; corrupted value?"
		p.fail()
	}
	if p.event.typ != e {
		p.parser.problem = fmt.Sprintf("expected %s event but got %s", e, p.event.typ)
		p.fail()
	}
	yaml_event_delete(&p.event)
	p.event.typ = yaml_NO_EVENT
}

// peek peeks at the next event in the event stream,
// puts the results into p.event and returns the event type.
func (p *parser) peek() yaml_event_type_t {
	if p.event.typ != yaml_NO_EVENT {
		return p.event.typ
	}
	// It's curious choice from the underlying API to generally return a
	// positive result on success, but on this case return true in an error
	// scenario. This was the source of bugs in the past (issue #666).
	if !yaml_parser_parse(&p.parser, &p.event) || p.parser.error != yaml_NO_ERROR {
		p.fail()
	}
	return p.event.typ
}

func (p *parser) fail() {
	if err := p.parser.read_error; err != nil {
		fail(err)
		return
	}

	var line, column int
	if p.parser.context_mark.line != 0 {
		line = p.parser.context_mark.line
		column = p.parser.context_mark.column
		// Scanner errors don't iterate line before returning error
		if p.parser.error == yaml_SCANNER_ERROR {
			line++
		}
	} else if p.parser.problem_mark.line != 0 {
		line = p.parser.problem_mark.line
		column = p.parser.problem_mark.column
		// Scanner errors don't iterate line before returning error
		if p.parser.error == yaml_SCANNER_ERROR {
			line++
		}
	}

	var offset int
	switch p.parser.error {
	case yaml_READER_ERROR:
		offset = p.parser.problem_offset
	case yaml_SCANNER_ERROR, yaml_PARSER_ERROR:
		offset = p.parser.problem_mark.index
	}

	var msg string
	if len(p.parser.problem) > 0 {
		msg = p.parser.problem
	} else {
		msg = "unknown problem parsing YAML content"
	}

	fail(syntaxErr(offset, line, column, msg))
}

func (p *parser) anchor(n *Node, anchor []byte) {
	if anchor != nil {
		n.Anchor = string(anchor)
		p.anchors[n.Anchor] = n
	}
}

func (p *parser) parse() *Node {
	p.init()
	switch p.peek() {
	case yaml_SCALAR_EVENT:
		return p.scalar()
	case yaml_ALIAS_EVENT:
		return p.alias()
	case yaml_MAPPING_START_EVENT:
		return p.mapping()
	case yaml_SEQUENCE_START_EVENT:
		return p.sequence()
	case yaml_DOCUMENT_START_EVENT:
		return p.document()
	case yaml_STREAM_END_EVENT:
		// Happens when attempting to decode an empty buffer.
		return nil
	case yaml_TAIL_COMMENT_EVENT:
		panic("internal error: unexpected tail comment event (please report)")
	default:
		panic("internal error: attempted to parse unknown event (please report): " + p.event.typ.String())
	}
}

func (p *parser) node(kind Kind, defaultTag, tag, value string) *Node {
	var style Style
	switch {
	case tag != "" && tag != "!":
		tag = shortTag(tag)
		style = TaggedStyle
	case defaultTag != "":
		tag = defaultTag
	case kind == ScalarNode:
		tag, _ = resolve("", value)
	}
	n := &Node{
		Kind:  kind,
		Tag:   tag,
		Value: value,
		Style: style,
	}
	if !p.textless {
		n.Line = p.event.start_mark.line + 1
		n.Column = p.event.start_mark.column + 1
		n.HeadComment = string(p.event.head_comment)
		n.LineComment = string(p.event.line_comment)
		n.FootComment = string(p.event.foot_comment)
	}
	return n
}

func (p *parser) parseChild(parent *Node) *Node {
	child := p.parse()
	parent.Content = append(parent.Content, child)
	return child
}

func (p *parser) document() *Node {
	n := p.node(DocumentNode, "", "", "")
	p.doc = n
	p.expect(yaml_DOCUMENT_START_EVENT)
	p.parseChild(n)
	if p.peek() == yaml_DOCUMENT_END_EVENT {
		n.FootComment = string(p.event.foot_comment)
	}
	p.expect(yaml_DOCUMENT_END_EVENT)
	return n
}

func (p *parser) alias() *Node {
	n := p.node(AliasNode, "", "", string(p.event.anchor))
	if _, ok := p.parentAnchors[n.Value]; ok {
		fail(unmarshalErrf(n, nil, "anchor %q value contains itself", n.Value))
	}
	n.Alias = p.anchors[n.Value]
	if n.Alias == nil {
		// FIXME: is that right error type?
		fail(unmarshalErrf(n, nil, "unknown anchor %q referenced", n.Value))
	}
	p.expect(yaml_ALIAS_EVENT)
	return n
}

func (p *parser) scalar() *Node {
	parsedStyle := p.event.scalar_style()
	var nodeStyle Style
	switch {
	case parsedStyle&yaml_DOUBLE_QUOTED_SCALAR_STYLE != 0:
		nodeStyle = DoubleQuotedStyle
	case parsedStyle&yaml_SINGLE_QUOTED_SCALAR_STYLE != 0:
		nodeStyle = SingleQuotedStyle
	case parsedStyle&yaml_LITERAL_SCALAR_STYLE != 0:
		nodeStyle = LiteralStyle
	case parsedStyle&yaml_FOLDED_SCALAR_STYLE != 0:
		nodeStyle = FoldedStyle
	}
	nodeValue := string(p.event.value)
	nodeTag := string(p.event.tag)
	var defaultTag string
	if nodeStyle == 0 {
		if nodeValue == "<<" {
			defaultTag = mergeTag
		}
	} else {
		defaultTag = strTag
	}
	n := p.node(ScalarNode, defaultTag, nodeTag, nodeValue)
	n.Style |= nodeStyle
	p.anchor(n, p.event.anchor)
	p.expect(yaml_SCALAR_EVENT)
	return n
}

func (p *parser) sequence() *Node {
	n := p.node(SequenceNode, seqTag, string(p.event.tag), "")
	if p.event.sequence_style()&yaml_FLOW_SEQUENCE_STYLE != 0 {
		n.Style |= FlowStyle
	}
	p.anchor(n, p.event.anchor)
	// Track the anchors of the parent nodes so that we can detect
	// recursive aliases.
	if anchor := n.Anchor; anchor != "" {
		p.parentAnchors[anchor] = struct{}{}
		defer func() {
			delete(p.parentAnchors, anchor)
		}()
	}
	p.expect(yaml_SEQUENCE_START_EVENT)
	for p.peek() != yaml_SEQUENCE_END_EVENT {
		p.parseChild(n)
	}
	n.LineComment = string(p.event.line_comment)
	n.FootComment = string(p.event.foot_comment)
	p.expect(yaml_SEQUENCE_END_EVENT)
	return n
}

func (p *parser) mapping() *Node {
	n := p.node(MappingNode, mapTag, string(p.event.tag), "")
	block := true
	if p.event.mapping_style()&yaml_FLOW_MAPPING_STYLE != 0 {
		block = false
		n.Style |= FlowStyle
	}
	p.anchor(n, p.event.anchor)
	// Track the anchors of the parent nodes so that we can detect
	// recursive aliases.
	if anchor := n.Anchor; anchor != "" {
		p.parentAnchors[anchor] = struct{}{}
		defer func() {
			delete(p.parentAnchors, anchor)
		}()
	}
	p.expect(yaml_MAPPING_START_EVENT)
	for p.peek() != yaml_MAPPING_END_EVENT {
		k := p.parseChild(n)
		if block && k.FootComment != "" {
			// Must be a foot comment for the prior value when being dedented.
			if len(n.Content) > 2 {
				n.Content[len(n.Content)-3].FootComment = k.FootComment
				k.FootComment = ""
			}
		}
		v := p.parseChild(n)
		if k.FootComment == "" && v.FootComment != "" {
			k.FootComment = v.FootComment
			v.FootComment = ""
		}
		if p.peek() == yaml_TAIL_COMMENT_EVENT {
			if k.FootComment == "" {
				k.FootComment = string(p.event.foot_comment)
			}
			p.expect(yaml_TAIL_COMMENT_EVENT)
		}
	}
	n.LineComment = string(p.event.line_comment)
	n.FootComment = string(p.event.foot_comment)
	if n.Style&FlowStyle == 0 && n.FootComment != "" && len(n.Content) > 1 {
		n.Content[len(n.Content)-2].FootComment = n.FootComment
		n.FootComment = ""
	}
	p.expect(yaml_MAPPING_END_EVENT)
	return n
}

// ----------------------------------------------------------------------------
// Decoder, unmarshals a node into a provided value.

type decoder struct {
	doc     *Node
	terrors []error

	stringMapType  reflect.Type
	generalMapType reflect.Type

	knownFields bool
	uniqueKeys  bool
	decodeCount int
	aliasCount  int
	aliasDepth  int

	mergedFields map[any]struct{}
}

var (
	nodeType       = reflect.TypeOf(Node{})
	ptrNodeType    = reflect.TypeOf(&Node{})
	durationType   = reflect.TypeOf(time.Duration(0))
	stringMapType  = reflect.TypeOf(map[string]any{})
	generalMapType = reflect.TypeOf(map[any]any{})
	ifaceType      = generalMapType.Elem()
)

func newDecoder() *decoder {
	d := &decoder{
		stringMapType:  stringMapType,
		generalMapType: generalMapType,
		uniqueKeys:     true,
	}
	return d
}

func (d *decoder) terror(n *Node, tag string, out reflect.Value) {
	if n.Tag != "" {
		tag = n.Tag
	}
	value := n.Value
	if tag != seqTag && tag != mapTag {
		if len(value) > 10 {
			value = " `" + value[:7] + "...`"
		} else {
			value = " `" + value + "`"
		}
	}

	typ := out.Type()
	d.terrors = append(d.terrors,
		unmarshalErrf(n, typ, "cannot unmarshal %s%s into %s", shortTag(tag), value, typ),
	)
}

func (d *decoder) mapCustomError(err error) bool {
	var e *TypeError
	if errors.As(err, &e) {
		d.terrors = append(d.terrors, multierr.Errors(e.Group)...)
		return false
	}
	if err != nil {
		fail(err)
	}
	return true
}

func (d *decoder) callUnmarshaler(n *Node, u Unmarshaler) (good bool) {
	return d.mapCustomError(u.UnmarshalYAML(n))
}

func (d *decoder) callObsoleteUnmarshaler(n *Node, u obsoleteUnmarshaler) (good bool) {
	terrlen := len(d.terrors)
	err := u.UnmarshalYAML(func(v any) (err error) {
		defer handleErr(&err)
		d.unmarshal(n, reflect.ValueOf(v))
		if len(d.terrors) > terrlen {
			issues := d.terrors[terrlen:]
			d.terrors = d.terrors[:terrlen]
			return &TypeError{
				Group: multierr.Combine(issues...),
			}
		}
		return nil
	})
	return d.mapCustomError(err)
}

// d.prepare initializes and dereferences pointers and calls UnmarshalYAML
// if a value is found to implement it.
// It returns the initialized and dereferenced out value, whether
// unmarshaling was already done by UnmarshalYAML, and if so whether
// its types unmarshaled appropriately.
//
// If n holds a null value, prepare returns before doing anything.
func (d *decoder) prepare(n *Node, out reflect.Value) (newout reflect.Value, unmarshaled, good bool) {
	isNull := n.ShortTag() == nullTag
	again := true
	for again {
		again = false
		if out.Kind() == reflect.Ptr {
			if isNull {
				// If the value is a null, don't initialize it.
				return out, false, false
			}
			if out.IsNil() {
				out.Set(reflect.New(out.Type().Elem()))
			}
			out = out.Elem()
			again = true
		}
		if out.CanAddr() {
			outi := out.Addr().Interface()
			if u, ok := outi.(Unmarshaler); ok {
				good = d.callUnmarshaler(n, u)
				return out, true, good
			}
			if u, ok := outi.(obsoleteUnmarshaler); ok {
				good = d.callObsoleteUnmarshaler(n, u)
				return out, true, good
			}
		}
	}
	return out, false, false
}

func (d *decoder) fieldByIndex(n *Node, v reflect.Value, index []int) (field reflect.Value) {
	if n.ShortTag() == nullTag {
		return reflect.Value{}
	}
	for _, num := range index {
		for {
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					v.Set(reflect.New(v.Type().Elem()))
				}
				v = v.Elem()
				continue
			}
			break
		}
		v = v.Field(num)
	}
	return v
}

const (
	// 400,000 decode operations is ~500kb of dense object declarations, or
	// ~5kb of dense object declarations with 10000% alias expansion
	aliasRatioRangeLow = 400000

	// 4,000,000 decode operations is ~5MB of dense object declarations, or
	// ~4.5MB of dense object declarations with 10% alias expansion
	aliasRatioRangeHigh = 4000000

	// aliasRatioRange is the range over which we scale allowed alias ratios
	aliasRatioRange = float64(aliasRatioRangeHigh - aliasRatioRangeLow)
)

func allowedAliasRatio(decodeCount int) float64 {
	switch {
	case decodeCount <= aliasRatioRangeLow:
		// allow 99% to come from alias expansion for small-to-medium documents
		return 0.99
	case decodeCount >= aliasRatioRangeHigh:
		// allow 10% to come from alias expansion for very large documents
		return 0.10
	default:
		// scale smoothly from 99% down to 10% over the range.
		// this maps to 396,000 - 400,000 allowed alias-driven decodes over the range.
		// 400,000 decode operations is ~100MB of allocations in worst-case scenarios (single-item maps).
		return 0.99 - 0.89*(float64(decodeCount-aliasRatioRangeLow)/aliasRatioRange)
	}
}

func (d *decoder) unmarshal(n *Node, out reflect.Value) (good bool) {
	d.decodeCount++
	if d.aliasDepth > 0 {
		d.aliasCount++
	}
	if d.aliasCount > 100 && d.decodeCount > 1000 && float64(d.aliasCount)/float64(d.decodeCount) > allowedAliasRatio(d.decodeCount) {
		fail(unmarshalErrf(n, out.Type(), "document contains excessive aliasing"))
	}
	switch out.Type() {
	case nodeType:
		out.Set(reflect.ValueOf(n).Elem())
		return true
	case ptrNodeType:
		out.Set(reflect.ValueOf(n))
		return true
	}
	switch n.Kind {
	case DocumentNode:
		return d.document(n, out)
	case AliasNode:
		return d.alias(n, out)
	}
	out, unmarshaled, good := d.prepare(n, out)
	if unmarshaled {
		return good
	}
	switch n.Kind {
	case ScalarNode:
		good = d.scalar(n, out)
	case MappingNode:
		good = d.mapping(n, out)
	case SequenceNode:
		good = d.sequence(n, out)
	case 0:
		if n.IsZero() {
			return d.null(out)
		}
		fallthrough
	default:
		fail(unmarshalErrf(n, out.Type(), "cannot decode node with unknown kind %d", n.Kind))
	}
	return good
}

func (d *decoder) document(n *Node, out reflect.Value) (good bool) {
	if len(n.Content) == 1 {
		d.doc = n
		d.unmarshal(n.Content[0], out)
		return true
	}
	return false
}

func (d *decoder) alias(n *Node, out reflect.Value) (good bool) {
	d.aliasDepth++
	good = d.unmarshal(n.Alias, out)
	d.aliasDepth--
	return good
}

func (d *decoder) null(out reflect.Value) bool {
	if out.CanAddr() {
		switch out.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
			out.Set(reflect.Zero(out.Type()))
			return true
		}
	}
	return false
}

func (d *decoder) scalar(n *Node, out reflect.Value) bool {
	var tag string
	var resolved any
	if n.indicatedString() {
		tag = strTag
		resolved = n.Value
	} else {
		tag, resolved = resolve(n.Tag, n.Value)
		if tag == binaryTag {
			data, err := base64.StdEncoding.DecodeString(resolved.(string))
			if err != nil {
				fail(unmarshalErrf(n, out.Type(), "decode !!binary: %w", err))
			}
			resolved = string(data)
		}
	}
	if resolved == nil {
		return d.null(out)
	}
	if resolvedv := reflect.ValueOf(resolved); out.Type() == resolvedv.Type() {
		// We've resolved to exactly the type we want, so use that.
		out.Set(resolvedv)
		return true
	}
	// Perhaps we can use the value as a TextUnmarshaler to
	// set its value.
	if out.CanAddr() {
		u, ok := out.Addr().Interface().(encoding.TextUnmarshaler)
		if ok {
			var text []byte
			if tag == binaryTag {
				text = []byte(resolved.(string))
			} else {
				// We let any value be unmarshaled into TextUnmarshaler.
				// That might be more lax than we'd like, but the
				// TextUnmarshaler itself should bowl out any dubious values.
				text = []byte(n.Value)
			}
			err := u.UnmarshalText(text)
			if err != nil {
				fail(err)
			}
			return true
		}
	}
	switch out.Kind() {
	case reflect.String:
		if tag == binaryTag {
			out.SetString(resolved.(string))
			return true
		}
		out.SetString(n.Value)
		return true
	case reflect.Interface:
		out.Set(reflect.ValueOf(resolved))
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// This used to work in v2, but it's very unfriendly.
		isDuration := out.Type() == durationType

		switch resolved := resolved.(type) {
		case int:
			if !isDuration && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				return true
			}
		case int64:
			if !isDuration && !out.OverflowInt(resolved) {
				out.SetInt(resolved)
				return true
			}
		case uint64:
			if !isDuration && resolved <= math.MaxInt64 && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				return true
			}
		case float64:
			if !isDuration && resolved <= math.MaxInt64 && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				return true
			}
		case string:
			if out.Type() == durationType {
				d, err := time.ParseDuration(resolved)
				if err == nil {
					out.SetInt(int64(d))
					return true
				}
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		switch resolved := resolved.(type) {
		case int:
			if resolved >= 0 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				return true
			}
		case int64:
			if resolved >= 0 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				return true
			}
		case uint64:
			if !out.OverflowUint(resolved) {
				out.SetUint(resolved)
				return true
			}
		case float64:
			if resolved <= math.MaxUint64 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				return true
			}
		}
	case reflect.Bool:
		switch resolved := resolved.(type) {
		case bool:
			out.SetBool(resolved)
			return true
		case string:
			// This offers some compatibility with the 1.1 spec (https://yaml.org/type/bool.html).
			// It only works if explicitly attempting to unmarshal into a typed bool value.
			switch resolved {
			case "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
				out.SetBool(true)
				return true
			case "n", "N", "no", "No", "NO", "off", "Off", "OFF":
				out.SetBool(false)
				return true
			}
		}
	case reflect.Float32, reflect.Float64:
		switch resolved := resolved.(type) {
		case int:
			out.SetFloat(float64(resolved))
			return true
		case int64:
			out.SetFloat(float64(resolved))
			return true
		case uint64:
			out.SetFloat(float64(resolved))
			return true
		case float64:
			out.SetFloat(resolved)
			return true
		}
	case reflect.Struct:
		if resolvedv := reflect.ValueOf(resolved); out.Type() == resolvedv.Type() {
			out.Set(resolvedv)
			return true
		}
	case reflect.Ptr:
		panic("yaml internal error: please report the issue")
	}
	d.terror(n, tag, out)
	return false
}

func settableValueOf(i any) reflect.Value {
	v := reflect.ValueOf(i)
	sv := reflect.New(v.Type()).Elem()
	sv.Set(v)
	return sv
}

func (d *decoder) sequence(n *Node, out reflect.Value) (good bool) {
	l := len(n.Content)

	var iface reflect.Value
	switch out.Kind() {
	case reflect.Slice:
		out.Set(reflect.MakeSlice(out.Type(), l, l))
	case reflect.Array:
		if l != out.Len() {
			fail(unmarshalErrf(n, out.Type(), "invalid array: want %d elements but got %d", out.Len(), l))
		}
	case reflect.Interface:
		// No type hints. Will have to use a generic sequence.
		iface = out
		out = settableValueOf(make([]any, l))
	default:
		d.terror(n, seqTag, out)
		return false
	}
	et := out.Type().Elem()

	j := 0
	for i := 0; i < l; i++ {
		e := reflect.New(et).Elem()
		if ok := d.unmarshal(n.Content[i], e); ok {
			out.Index(j).Set(e)
			j++
		}
	}
	if out.Kind() != reflect.Array {
		out.Set(out.Slice(0, j))
	}
	if iface.IsValid() {
		iface.Set(out)
	}
	return true
}

func failWantHashable(n *Node, val reflect.Value) {
	fail(unmarshalErrf(n, val.Type(), "invalid map key: %#v", val.Interface()))
}

func (d *decoder) mapping(n *Node, out reflect.Value) (good bool) {
	l := len(n.Content)
	if d.uniqueKeys {
		nerrs := len(d.terrors)
		for i := 0; i < l; i += 2 {
			ni := n.Content[i]
			for j := i + 2; j < l; j += 2 {
				nj := n.Content[j]
				if ni.equalKey(nj) {
					d.terrors = append(d.terrors, duplicateKeyErr(nj, ni, out.Type()))
				}
			}
		}
		if len(d.terrors) > nerrs {
			return false
		}
	}
	switch out.Kind() {
	case reflect.Struct:
		return d.mappingStruct(n, out)
	case reflect.Map:
		// okay
	case reflect.Interface:
		iface := out
		if isStringMap(n) {
			out = reflect.MakeMap(d.stringMapType)
		} else {
			out = reflect.MakeMap(d.generalMapType)
		}
		iface.Set(out)
	default:
		d.terror(n, mapTag, out)
		return false
	}

	outt := out.Type()
	kt := outt.Key()
	et := outt.Elem()

	stringMapType := d.stringMapType
	generalMapType := d.generalMapType
	if outt.Elem() == ifaceType {
		if outt.Key().Kind() == reflect.String {
			d.stringMapType = outt
		} else if outt.Key() == ifaceType {
			d.generalMapType = outt
		}
	}

	mergedFields := d.mergedFields
	d.mergedFields = nil

	var mergeNode *Node

	mapIsNew := false
	if out.IsNil() {
		out.Set(reflect.MakeMap(outt))
		mapIsNew = true
	}
	for i := 0; i < l; i += 2 {
		if isMerge(n.Content[i]) {
			mergeNode = n.Content[i+1]
			continue
		}
		k := reflect.New(kt).Elem()
		if d.unmarshal(n.Content[i], k) {
			if !isHashable(k) {
				failWantHashable(n.Content[i], k)
				return
			}
			if mergedFields != nil {
				ki := k.Interface()
				if _, ok := mergedFields[ki]; ok {
					continue
				}
				mergedFields[ki] = struct{}{}
			}
			e := reflect.New(et).Elem()
			if d.unmarshal(n.Content[i+1], e) || n.Content[i+1].ShortTag() == nullTag && (mapIsNew || !out.MapIndex(k).IsValid()) {
				out.SetMapIndex(k, e)
			}
		}
	}

	d.mergedFields = mergedFields
	if mergeNode != nil {
		d.merge(n, mergeNode, out)
	}

	d.stringMapType = stringMapType
	d.generalMapType = generalMapType
	return true
}

func isStringMap(n *Node) bool {
	if n.Kind != MappingNode {
		return false
	}
	l := len(n.Content)
	for i := 0; i < l; i += 2 {
		shortTag := n.Content[i].ShortTag()
		if shortTag != strTag && shortTag != mergeTag {
			return false
		}
	}
	return true
}

func (d *decoder) mappingStruct(n *Node, out reflect.Value) (good bool) {
	sinfo, err := getStructInfo(out.Type())
	if err != nil {
		panic(err)
	}

	var inlineMap reflect.Value
	var elemType reflect.Type
	if sinfo.InlineMap != -1 {
		inlineMap = out.Field(sinfo.InlineMap)
		elemType = inlineMap.Type().Elem()
	}

	for _, index := range sinfo.InlineUnmarshalers {
		field := d.fieldByIndex(n, out, index)
		d.prepare(n, field)
	}

	mergedFields := d.mergedFields
	d.mergedFields = nil
	var mergeNode *Node
	var doneFields []bool
	if d.uniqueKeys {
		doneFields = make([]bool, len(sinfo.FieldsList))
	}
	name := settableValueOf("")
	l := len(n.Content)
	for i := 0; i < l; i += 2 {
		ni := n.Content[i]
		if isMerge(ni) {
			mergeNode = n.Content[i+1]
			continue
		}
		if !d.unmarshal(ni, name) {
			continue
		}
		sname := name.String()
		if mergedFields != nil {
			if _, ok := mergedFields[sname]; ok {
				continue
			}
			mergedFields[sname] = struct{}{}
		}

		switch info, ok := sinfo.FieldsMap[sname]; {
		case ok:
			if d.uniqueKeys {
				if doneFields[info.ID] {
					// TODO(tdakkota): find second occurrence?
					d.terrors = append(d.terrors, duplicateKeyErr(ni, nil, out.Type()))
					continue
				}
				doneFields[info.ID] = true
			}
			var field reflect.Value
			if info.Inline == nil {
				field = out.Field(info.Num)
			} else {
				field = d.fieldByIndex(n, out, info.Inline)
			}
			d.unmarshal(n.Content[i+1], field)
		case sinfo.InlineMap != -1:
			if inlineMap.IsNil() {
				inlineMap.Set(reflect.MakeMap(inlineMap.Type()))
			}
			value := reflect.New(elemType).Elem()
			d.unmarshal(n.Content[i+1], value)
			inlineMap.SetMapIndex(name, value)
		case d.knownFields:
			d.terrors = append(d.terrors, unknownFieldErr(name.String(), ni, out.Type()))
		}
	}

	d.mergedFields = mergedFields
	if mergeNode != nil {
		d.merge(n, mergeNode, out)
	}
	return true
}

func failWantMap(merge *Node, typ reflect.Type) {
	fail(unmarshalErrf(merge, typ, "map merge requires map or sequence of maps as the value"))
}

func (d *decoder) merge(parent, merge *Node, out reflect.Value) {
	mergedFields := d.mergedFields
	if mergedFields == nil {
		d.mergedFields = make(map[any]struct{})
		for i := 0; i < len(parent.Content); i += 2 {
			k := reflect.New(ifaceType).Elem()
			if n := parent.Content[i]; d.unmarshal(n, k) {
				if !isHashable(k) {
					failWantHashable(n, k)
					return
				}
				d.mergedFields[k.Interface()] = struct{}{}
			}
		}
	}

	switch merge.Kind {
	case MappingNode:
		d.unmarshal(merge, out)
	case AliasNode:
		if a := merge.Alias; a != nil && a.Kind != MappingNode {
			failWantMap(a, out.Type())
		}
		d.unmarshal(merge, out)
	case SequenceNode:
		for i := 0; i < len(merge.Content); i++ {
			ni := merge.Content[i]
			if ni.Kind == AliasNode {
				if a := ni.Alias; a != nil && a.Kind != MappingNode {
					failWantMap(a, out.Type())
				}
			} else if ni.Kind != MappingNode {
				failWantMap(ni, out.Type())
			}
			d.unmarshal(ni, out)
		}
	default:
		failWantMap(merge, out.Type())
	}

	d.mergedFields = mergedFields
}

func isMerge(n *Node) bool {
	return n.Kind == ScalarNode && n.Value == "<<" && (n.Tag == "" || n.Tag == "!" || shortTag(n.Tag) == mergeTag)
}
