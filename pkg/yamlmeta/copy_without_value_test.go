// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package yamlmeta_test

import (
	"testing"

	"carvel.dev/ytt/pkg/yamlmeta"
	"github.com/stretchr/testify/require"
)

// testAnnotations stands in for template annotations, which nodes copy with
// DeepCopyAsInterface.
type testAnnotations map[string]string

func (a testAnnotations) DeepCopyAsInterface() any {
	result := testAnnotations{}
	for k, v := range a {
		result[k] = v
	}
	return result
}

type valuelessCopier interface {
	CopyWithoutValueAsInterface() any
}

// Every node type copied without its value equals the node deep-copied and
// then reset.
func TestCopyWithoutValueMatchesDeepCopyAndReset(t *testing.T) {
	const data = `#! set comment
--- #! doc comment
map: #! map item comment
  key: value
array:
- item #! array item comment
- nested:
  - 1
---
- top
#! trailing comment, kept in an injected document
`
	opts := yamlmeta.ParserOpts{WithoutComments: false}
	docSet, err := yamlmeta.NewParser(opts).ParseBytes([]byte(data), "t.yml")
	require.NoError(t, err)

	seen := map[string]bool{}
	var check func(node yamlmeta.Node)
	check = func(node yamlmeta.Node) {
		node.SetAnnotations(testAnnotations{"node": "annotation"})

		copier, ok := node.(valuelessCopier)
		require.True(t, ok, "%T does not implement CopyWithoutValueAsInterface",
			node)

		expected := node.DeepCopyAsNode()
		expected.ResetValue()
		require.Equal(t, expected, copier.CopyWithoutValueAsInterface())

		seen[typeName(node)] = true
		for _, child := range node.GetValues() {
			if childNode, ok := child.(yamlmeta.Node); ok {
				check(childNode)
			}
		}
	}
	check(docSet)

	require.Equal(t, map[string]bool{
		"DocumentSet": true, "Document": true, "Map": true,
		"MapItem": true, "Array": true, "ArrayItem": true,
	}, seen)
}

func typeName(node yamlmeta.Node) string {
	switch node.(type) {
	case *yamlmeta.DocumentSet:
		return "DocumentSet"
	case *yamlmeta.Document:
		return "Document"
	case *yamlmeta.Map:
		return "Map"
	case *yamlmeta.MapItem:
		return "MapItem"
	case *yamlmeta.Array:
		return "Array"
	case *yamlmeta.ArrayItem:
		return "ArrayItem"
	default:
		return "unknown"
	}
}
