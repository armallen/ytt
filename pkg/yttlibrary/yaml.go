// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package yttlibrary

import (
	"fmt"
	"sync"

	"carvel.dev/ytt/pkg/template/core"
	"carvel.dev/ytt/pkg/yamlmeta"
	"github.com/k14s/starlark-go/starlark"
	"github.com/k14s/starlark-go/starlarkstruct"
)

var (
	// YAMLAPI contains the definition of the @ytt:yaml module
	YAMLAPI = starlark.StringDict{
		"yaml": &starlarkstruct.Module{
			Name: "yaml",
			Members: starlark.StringDict{
				"encode": starlark.NewBuiltin("yaml.encode", core.ErrWrapper(yamlModule{}.starlarkEncode)),
				"decode": starlark.NewBuiltin("yaml.decode", core.ErrWrapper(yamlModule{}.starlarkDecode)),
			},
		},
	}
)

type yamlModule struct{}

// starlarkEncode adapts a call from Starlark to yamlModule.Encode()
func (b yamlModule) starlarkEncode(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if args.Len() != 1 {
		return starlark.None, fmt.Errorf("expected exactly one argument")
	}

	val, err := core.NewStarlarkValue(args.Index(0)).AsGoValue()
	if err != nil {
		return starlark.None, err
	}

	encoded, err := b.Encode(val)
	if err != nil {
		return starlark.None, nil
	}

	return starlark.String(encoded), nil
}

// Encode renders the provided input as a YAML-formatted string
func (b yamlModule) Encode(goValue interface{}) (string, error) {
	var docSet *yamlmeta.DocumentSet

	switch typedVal := goValue.(type) {
	case *yamlmeta.DocumentSet:
		docSet = typedVal
	case *yamlmeta.Document:
		// Documents should be part of DocumentSet by the time it makes it here
		panic("Unexpected document")
	default:
		docSet = &yamlmeta.DocumentSet{Items: []*yamlmeta.Document{{Value: typedVal}}}
	}

	valBs, err := docSet.AsBytes()
	if err != nil {
		return "", err
	}

	return string(valBs), nil
}

// Decode is a core.StarlarkFunc that parses the provided input from YAML format into dicts, lists, and scalars
func (b yamlModule) starlarkDecode(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if args.Len() != 1 {
		return starlark.None, fmt.Errorf("expected exactly one argument")
	}

	valEncoded, err := core.NewStarlarkValue(args.Index(0)).AsString()
	if err != nil {
		return starlark.None, err
	}

	value, err := b.Decode(valEncoded)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (b yamlModule) Decode(yamlString string) (starlark.Value, error) {
	decoded, err := decodeCached(yamlString)
	if err != nil {
		return starlark.None, err
	}

	value := core.NewGoValue(decoded).AsStarlarkValue()
	return value, nil
}

const (
	decodeCacheMinLen   = 1 << 10   // smaller inputs are cheap to parse
	decodeCacheMaxBytes = 256 << 20 // total size of cached inputs
)

// decodeCache memoizes the parse of large YAML strings. Templates commonly
// decode the same data file (via data.read) in every evaluation. The parsed
// values are never modified: each Decode converts them into new Starlark
// values, which callers may mutate.
var decodeCache = struct {
	sync.Mutex
	entries map[string]interface{}
	bytes   int
}{entries: map[string]interface{}{}}

func decodeCached(yamlString string) (interface{}, error) {
	cacheable := len(yamlString) >= decodeCacheMinLen
	if cacheable {
		decodeCache.Lock()
		decoded, ok := decodeCache.entries[yamlString]
		decodeCache.Unlock()
		if ok {
			return decoded, nil
		}
	}

	var decoded interface{}
	if err := yamlmeta.PlainUnmarshal([]byte(yamlString), &decoded); err != nil {
		return nil, err
	}

	if cacheable {
		decodeCache.Lock()
		if _, ok := decodeCache.entries[yamlString]; !ok && decodeCache.bytes+len(yamlString) <= decodeCacheMaxBytes {
			decodeCache.entries[yamlString] = decoded
			decodeCache.bytes += len(yamlString)
		}
		decodeCache.Unlock()
	}

	return decoded, nil
}
