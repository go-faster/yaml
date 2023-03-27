package yaml_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-faster/jx"

	"github.com/go-faster/yaml"
)

func TestNode_EncodeJSON(t *testing.T) {
	type testCase struct {
		input   yaml.Node
		output  string
		wantErr bool
	}
	mustNode := func(input string) (n yaml.Node) {
		require.NoError(t, yaml.Unmarshal([]byte(input), &n))
		return n
	}

	tests := []testCase{
		{mustNode(`foobar`), `"foobar"`, false},
		{mustNode(`"\nfoo\n"`), `"\nfoo\n"`, false},
		{mustNode(`10`), `10`, false},
		{mustNode(`null`), `null`, false},

		{yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str"}, `""`, false},
		{yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "10"}, `10`, false},
		{yaml.Node{Kind: yaml.SequenceNode}, "[]", false},
		{yaml.Node{Kind: yaml.MappingNode}, "{}", false},
		{
			yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Anchor: "a", Value: "key"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Anchor: "b", Value: "value"},
					{Kind: yaml.AliasNode, Alias: &yaml.Node{
						Kind:   yaml.ScalarNode,
						Tag:    "!!str",
						Anchor: "b",
						Value:  "value",
					}},
					{Kind: yaml.AliasNode, Alias: &yaml.Node{
						Kind:   yaml.ScalarNode,
						Tag:    "!!str",
						Anchor: "a",
						Value:  "key",
					}},
				},
			},
			`{"key": "value", "value": "key"}`,
			false,
		},
		{
			yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Tag: "!!str", Anchor: "a", Value: "a"},
							{Kind: yaml.ScalarNode, Tag: "!!str", Anchor: "b", Value: "b"},
							{Kind: yaml.AliasNode, Alias: &yaml.Node{
								Kind:   yaml.ScalarNode,
								Tag:    "!!str",
								Anchor: "a",
								Value:  "a",
							}},
							{Kind: yaml.AliasNode, Alias: &yaml.Node{
								Kind:   yaml.ScalarNode,
								Tag:    "!!str",
								Anchor: "b",
								Value:  "b",
							}},
						},
					},
				},
			},
			`["a","b","a","b"]`,
			false,
		},

		{
			yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!int", Value: "10"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "foo"},
				},
			},
			"",
			true,
		},
		{yaml.Node{Kind: yaml.DocumentNode}, "", true},
		{
			yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{}, {},
				},
			},
			"",
			true,
		},
		{yaml.Node{}, "", true},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			a := require.New(t)

			var e jx.Encoder
			err := tt.input.EncodeJSON(&e)
			if tt.wantErr {
				a.Error(err)
				return
			}
			a.NoError(err)
			a.JSONEq(tt.output, e.String())
		})
	}
}
