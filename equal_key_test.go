package yaml

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNode_equalKey(t *testing.T) {
	mustNode := func(s string) *Node {
		var n Node
		require.NoError(t, Unmarshal([]byte(s), &n))
		if n.Kind == DocumentNode {
			return n.Content[0]
		}
		return &n
	}
	theNode := mustNode(`a: 1`)

	tests := []struct {
		a, b *Node
		want bool
	}{
		// Same node.
		{theNode, theNode, true},

		// Nil nodes.
		{nil, nil, false},
		{nil, mustNode("null"), false},
		{mustNode("null"), nil, false},

		// Different kinds.
		{mustNode("{}"), mustNode("[]"), false},
		{mustNode("{}"), mustNode("true"), false},
		{mustNode("[]"), mustNode("true"), false},
		{mustNode("{}: 1"), mustNode("a: 1"), false},
		{mustNode("{{}: 1}: 1"), mustNode("{a: 1}: 1"), false},
		{mustNode("{{}: 1, b: 2}: 1"), mustNode("{a: 1, b: 2}: 1"), false},

		// Scalars.
		{mustNode("null"), mustNode("null"), true},
		{mustNode("true"), mustNode("true"), true},
		{mustNode("false"), mustNode("false"), true},
		{mustNode("0"), mustNode("0"), true},
		{mustNode("1"), mustNode("1"), true},
		{mustNode("foo"), mustNode("foo"), true},
		{mustNode(`"f_\u000a_oo"`), mustNode(`"f_\n_oo"`), true},

		{mustNode("true"), mustNode("false"), false},
		{mustNode("null"), mustNode("false"), false},
		{mustNode("1"), mustNode("0"), false},
		{mustNode("baz"), mustNode("foo"), false},

		// Arrays.
		{mustNode("[]"), mustNode("[]"), true},
		{mustNode("[0]"), mustNode("[0]"), true},
		{mustNode("- 0"), mustNode("[0]"), true},
		{mustNode("[0, 1]"), mustNode("[0, 1]"), true},

		{mustNode("[0]"), mustNode("[1]"), false},
		{mustNode("[0, 1]"), mustNode("[0]"), false},

		// Objects.
		{mustNode("{}"), mustNode("{}"), true},
		{mustNode("a: 1"), mustNode("a: 1"), true},
		{mustNode(`{"a": 1}`), mustNode("a: 1"), true},
		{mustNode("a: 1"), mustNode("a: 1 # comment"), true},
		{mustNode("a: 1\nb: 1"), mustNode("b: 1\na: 1"), true},
		{mustNode("a: 1\nc: [{}]\nb: 1"), mustNode("b: 1\nc: [{}]\na: 1"), true},

		{mustNode("a: 1"), mustNode("a: 2"), false},
		{mustNode("a: 1"), mustNode("b: 1"), false},
		{mustNode("a: 1\nb: 1"), mustNode("b: 1"), false},
		{mustNode("a: 1\nb: 1\nc: 1"), mustNode("a: 1\nb: 1\nc: 2"), false},

		// Objects with complex keys.
		{mustNode("[]: 1"), mustNode("[]: 1"), true},
		{mustNode("{}: 1"), mustNode("{}: 1"), true},
		{mustNode("{a: 1}: 1"), mustNode("{a: 1}: 1"), true},
		{mustNode("{b: 1, a: 1}: 1"), mustNode("{a: 1, b: 1}: 1"), true},

		{mustNode("{a: 1}: 1"), mustNode("{a: 2}: 1"), false},
		{mustNode("{a: 1, b: 2}: 1"), mustNode("{a: 1}: 1"), false},
		{mustNode("{b: 1, a: 1}: 1"), mustNode("{a: 1, b: []}: 1"), false},

		// Canonical representation. Currently not supported, but should be.
		// !int
		{mustNode("10"), mustNode("+10"), false},
		{mustNode("10"), mustNode("0xa"), false},
		{mustNode("10"), mustNode("012"), false},
		{mustNode("10"), mustNode("0b1010"), false},
		{mustNode("0xA"), mustNode("0xa"), false},
		// !!float
		{mustNode("10"), mustNode("10.0"), false},
		{mustNode("10"), mustNode("1e1"), false},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			a := require.New(t)

			check := a.False
			if tt.want {
				check = a.True
			}
			// Ensure that equality is symmetric relation.
			check(tt.a.equalKey(tt.b))
			check(tt.b.equalKey(tt.a))
		})
	}
}
