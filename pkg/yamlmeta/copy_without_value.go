// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package yamlmeta

// CopyWithoutValueAsInterface returns a copy of the node equivalent to
// DeepCopy() followed by ResetValue(), without copying the discarded subtree.

func (ds *DocumentSet) CopyWithoutValueAsInterface() interface{} {
	return &DocumentSet{
		Comments:    []*Comment(CommentSlice(ds.Comments).DeepCopy()),
		AllComments: []*Comment(CommentSlice(ds.AllComments).DeepCopy()),
		Position:    ds.Position,
		annotations: annotationsDeepCopy(ds.annotations),
	}
}

func (d *Document) CopyWithoutValueAsInterface() interface{} {
	return &Document{
		Comments:    []*Comment(CommentSlice(d.Comments).DeepCopy()),
		Position:    d.Position,
		annotations: annotationsDeepCopy(d.annotations),
		injected:    d.injected,
	}
}

func (m *Map) CopyWithoutValueAsInterface() interface{} {
	return &Map{
		Comments:    []*Comment(CommentSlice(m.Comments).DeepCopy()),
		Position:    m.Position,
		annotations: annotationsDeepCopy(m.annotations),
	}
}

func (mi *MapItem) CopyWithoutValueAsInterface() interface{} {
	return &MapItem{
		Comments:    []*Comment(CommentSlice(mi.Comments).DeepCopy()),
		Key:         mi.Key,
		Position:    mi.Position,
		annotations: annotationsDeepCopy(mi.annotations),
	}
}

func (a *Array) CopyWithoutValueAsInterface() interface{} {
	return &Array{
		Comments:    []*Comment(CommentSlice(a.Comments).DeepCopy()),
		Position:    a.Position,
		annotations: annotationsDeepCopy(a.annotations),
	}
}

func (ai *ArrayItem) CopyWithoutValueAsInterface() interface{} {
	return &ArrayItem{
		Comments:    []*Comment(CommentSlice(ai.Comments).DeepCopy()),
		Position:    ai.Position,
		annotations: annotationsDeepCopy(ai.annotations),
	}
}
