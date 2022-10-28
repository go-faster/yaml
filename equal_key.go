package yaml

// equalKey returns true if the two nodes have the same key.
//
// Notice that this function doesn't fully implement the YAML spec comparison,
// defined here: https://yaml.org/spec/1.2.2/#node-comparison.
//
// The two main differences are:
//  1. The spec says to compare the tags, but this function only checks the kind. It's
//     because YAML tags are not explicit and the same value can have different tags.
//  2. The spec says to compare canonical values, but this function compares the
//     original values. It could be a problem if the original value is escaped or written in different style (10 vs 1e1).
//     But impact of this difference is very small, go-yaml compares the original values too.
func (n *Node) equalKey(b *Node) bool {
	if n == nil || b == nil {
		return false
	}
	if n.Kind != b.Kind {
		return false
	}
	switch n.Kind {
	case ScalarNode:
		// FIXME(tdakkota): compare canonical values.
		return n.Value == b.Value
	case SequenceNode:
		if len(n.Content) != len(b.Content) {
			return false
		}
		for i, n := range n.Content {
			if !n.equalKey(b.Content[i]) {
				return false
			}
		}
	case MappingNode:
		if len(n.Content) != len(b.Content) {
			return false
		}
		type nodeKey struct {
			Kind       Kind
			Value      string
			ContentLen int
		}
		type nodePair struct {
			Key *Node
			Val *Node
		}
		nodes := make(map[nodeKey][]nodePair, len(n.Content)/2)
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i]
			value := n.Content[i+1]
			nkey := nodeKey{key.Kind, key.Value, len(key.Content)}
			nodes[nkey] = append(nodes[nkey], nodePair{key, value})
		}
		for i := 0; i < len(b.Content); i += 2 {
			key := b.Content[i]
			value := b.Content[i+1]
			nkey := nodeKey{key.Kind, key.Value, len(key.Content)}

			similar, ok := nodes[nkey]
			if !ok {
				return false
			}
			for _, pair := range similar {
				if !pair.Key.equalKey(key) || !pair.Val.equalKey(value) {
					return false
				}
			}
		}
	case AliasNode:
		return n.Alias.equalKey(b.Alias)
	}
	return true
}
