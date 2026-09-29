// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package yttlibrary

import (
	"sync"

	"carvel.dev/ytt/pkg/yamlmeta"
	"github.com/k14s/starlark-go/starlark"
)

const (
	// yamlDecodeCacheMinLen is the length below which yaml.decode inputs are
	// parsed every time: they are cheap to parse.
	yamlDecodeCacheMinLen = 1 << 10

	yamlDecodeCacheKey = "ytt.yamlDecodeCache"
)

// YAMLDecodeCache keeps the values yaml.decode parsed from inputs of 1KiB or
// more, so decoding the same string again (e.g. the same data file in every
// evaluation) skips parsing. It holds at most maxBytes of input. yaml.decode
// still converts the cached value into new Starlark values on every call, so
// templates can change what it returns.
//
// A YAMLDecodeCache is safe for concurrent use. A nil *YAMLDecodeCache caches
// nothing.
type YAMLDecodeCache struct {
	mu       sync.Mutex
	entries  map[string]any
	bytes    int
	maxBytes int
}

// NewYAMLDecodeCache returns an empty cache holding at most maxBytes of
// yaml.decode input.
func NewYAMLDecodeCache(maxBytes int) *YAMLDecodeCache {
	return &YAMLDecodeCache{entries: map[string]any{}, maxBytes: maxBytes}
}

// SetYAMLDecodeCache makes yaml.decode use cache in thread.
func SetYAMLDecodeCache(thread *starlark.Thread, cache *YAMLDecodeCache) {
	thread.SetLocal(yamlDecodeCacheKey, cache)
}

// yamlDecodeCacheOf returns the cache set in thread, or nil.
func yamlDecodeCacheOf(thread *starlark.Thread) *YAMLDecodeCache {
	if thread == nil {
		return nil
	}
	if cache, ok := thread.Local(yamlDecodeCacheKey).(*YAMLDecodeCache); ok {
		return cache
	}
	return nil
}

// decode parses yamlString, or returns the value parsed from the same string
// before.
func (c *YAMLDecodeCache) decode(yamlString string) (any, error) {
	cacheable := c != nil && len(yamlString) >= yamlDecodeCacheMinLen
	if cacheable {
		if decoded, ok := c.lookup(yamlString); ok {
			return decoded, nil
		}
	}

	var decoded any
	err := yamlmeta.PlainUnmarshal([]byte(yamlString), &decoded)
	if err != nil {
		return nil, err
	}

	if cacheable {
		c.store(yamlString, decoded)
	}
	return decoded, nil
}

func (c *YAMLDecodeCache) lookup(yamlString string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	decoded, ok := c.entries[yamlString]
	return decoded, ok
}

// store keeps decoded unless yamlString is already cached or the cache is
// full.
func (c *YAMLDecodeCache) store(yamlString string, decoded any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, cached := c.entries[yamlString]
	if cached || c.bytes+len(yamlString) > c.maxBytes {
		return
	}
	c.entries[yamlString] = decoded
	c.bytes += len(yamlString)
}

// Len returns the number of inputs cached.
func (c *YAMLDecodeCache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
