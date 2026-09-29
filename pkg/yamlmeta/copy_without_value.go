// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package yamlmeta

// CopyWithoutValueAsInterface returns a copy of the DocumentSet equal to
// DeepCopy() followed by ResetValue(), without copying its items.
func (ds *DocumentSet) CopyWithoutValueAsInterface() any {
	return &DocumentSet{
		Comments:    []*Comment(CommentSlice(ds.Comments).DeepCopy()),
		AllComments: []*Comment(CommentSlice(ds.AllComments).DeepCopy()),
		Position:    ds.Position,
		annotations: annotationsDeepCopy(ds.annotations),
	}
}

// CopyWithoutValueAsInterface returns a copy of the Document equal to
// DeepCopy() followed by ResetValue(), without copying its value.
func (d *Document) CopyWithoutValueAsInterface() any {
	return &Document{
		Comments:    []*Comment(CommentSlice(d.Comments).DeepCopy()),
		Position:    d.Position,
		annotations: annotationsDeepCopy(d.annotations),
		injected:    d.injected,
	}
}

// CopyWithoutValueAsInterface returns a copy of the Map equal to
// DeepCopy() followed by ResetValue(), without copying its items.
func (m *Map) CopyWithoutValueAsInterface() any {
	return &Map{
		Comments:    []*Comment(CommentSlice(m.Comments).DeepCopy()),
		Position:    m.Position,
		annotations: annotationsDeepCopy(m.annotations),
	}
}

// CopyWithoutValueAsInterface returns a copy of the MapItem equal to
// DeepCopy() followed by ResetValue(), without copying its value.
func (mi *MapItem) CopyWithoutValueAsInterface() any {
	return &MapItem{
		Comments:    []*Comment(CommentSlice(mi.Comments).DeepCopy()),
		Key:         mi.Key,
		Position:    mi.Position,
		annotations: annotationsDeepCopy(mi.annotations),
	}
}

// CopyWithoutValueAsInterface returns a copy of the Array equal to
// DeepCopy() followed by ResetValue(), without copying its items.
func (a *Array) CopyWithoutValueAsInterface() any {
	return &Array{
		Comments:    []*Comment(CommentSlice(a.Comments).DeepCopy()),
		Position:    a.Position,
		annotations: annotationsDeepCopy(a.annotations),
	}
}

// CopyWithoutValueAsInterface returns a copy of the ArrayItem equal to
// DeepCopy() followed by ResetValue(), without copying its value.
func (ai *ArrayItem) CopyWithoutValueAsInterface() any {
	return &ArrayItem{
		Comments:    []*Comment(CommentSlice(ai.Comments).DeepCopy()),
		Position:    ai.Position,
		annotations: annotationsDeepCopy(ai.annotations),
	}
}
