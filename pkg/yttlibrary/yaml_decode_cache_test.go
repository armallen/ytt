// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package yttlibrary

import (
	"strings"
	"testing"

	"github.com/k14s/starlark-go/starlark"
)

const testItems = 2

// testYAML returns a cacheable YAML document with a list of testItems items.
func testYAML(name string) string {
	return "items: [1, 2]\n" + name + ": " +
		strings.Repeat("x", yamlDecodeCacheMinLen) + "\n"
}

// A cached decode returns new values each time, so changing one result does
// not change the next.
func TestYAMLDecodeCacheReturnsNewValues(t *testing.T) {
	cache := NewYAMLDecodeCache(1 << 20)
	thread := &starlark.Thread{}
	SetYAMLDecodeCache(thread, cache)
	input := testYAML("a")

	first := decodeItems(t, thread, input)
	if err := first.Append(starlark.MakeInt(testItems + 1)); err != nil {
		t.Fatalf("appending: %v", err)
	}

	second := decodeItems(t, thread, input)
	if second.Len() != testItems {
		t.Errorf("expected %d items after changing a previous result, got %s",
			testItems, second)
	}
	if cache.Len() != 1 {
		t.Errorf("expected 1 cached input, got %d", cache.Len())
	}
}

func TestYAMLDecodeCacheLimits(t *testing.T) {
	input := testYAML("a")

	// Without a cache, nothing is kept.
	decodeItems(t, &starlark.Thread{}, input)

	// Short inputs are not cached; inputs past maxBytes are not cached.
	cache := NewYAMLDecodeCache(len(input))
	thread := &starlark.Thread{}
	SetYAMLDecodeCache(thread, cache)
	decodeItems(t, thread, "items: [1, 2]\n")
	decodeItems(t, thread, input)
	decodeItems(t, thread, testYAML("b"))
	if cache.Len() != 1 {
		t.Errorf("expected 1 cached input, got %d", cache.Len())
	}
}

func decodeItems(t *testing.T, thread *starlark.Thread,
	input string,
) *starlark.List {
	t.Helper()
	decoded, err := yamlModule{}.starlarkDecode(thread, nil,
		starlark.Tuple{starlark.String(input)}, nil)
	if err != nil {
		t.Fatalf("decoding: %v", err)
	}
	dict, ok := decoded.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected a dict, got %T", decoded)
	}
	items, found, err := dict.Get(starlark.String("items"))
	if err != nil || !found {
		t.Fatalf("expected items in %s (%v)", dict, err)
	}
	list, ok := items.(*starlark.List)
	if !ok {
		t.Fatalf("expected a list, got %T", items)
	}
	return list
}
