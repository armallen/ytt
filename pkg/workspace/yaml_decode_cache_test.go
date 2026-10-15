// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package workspace_test

import (
	"strings"
	"testing"

	"carvel.dev/ytt/pkg/cmd/ui"
	"carvel.dev/ytt/pkg/files"
	"carvel.dev/ytt/pkg/workspace"
	"carvel.dev/ytt/pkg/workspace/datavalues"
	"carvel.dev/ytt/pkg/yttlibrary"
	"github.com/stretchr/testify/require"
)

// Templates evaluated by a factory with a YAMLDecodeCache, or by factories
// derived from it, decode through it; others do not.
func TestFactoryYAMLDecodeCache(t *testing.T) {
	cache := yttlibrary.NewYAMLDecodeCache(1 << 20)
	base := workspace.NewLibraryExecutionFactory(ui.NewTTY(false),
		workspace.TemplateLoaderOpts{}, false)
	withCache := base.WithYAMLDecodeCache(cache)
	ignoreComments := true

	factories := []*workspace.LibraryExecutionFactory{
		withCache,
		withCache.WithTemplateLoaderOptsOverrides(
			workspace.TemplateLoaderOptsOverrides{
				IgnoreUnknownComments: &ignoreComments,
			}),
		withCache.ThatSkipsDataValuesValidations(true),
	}
	for i, factory := range factories {
		evalDecode(t, factory, string(rune('a'+i)))
		require.Equal(t, i+1, cache.Len(),
			"factory %d did not use the cache", i)
	}

	evalDecode(t, base, "z")
	require.Equal(t, len(factories), cache.Len(),
		"a factory without the cache used it")
}

// evalDecode evaluates a template decoding a data file of 1KiB or more,
// unique to name.
func evalDecode(t *testing.T, factory *workspace.LibraryExecutionFactory,
	name string,
) {
	t.Helper()
	table := "name: " + name + "\npadding: " + strings.Repeat("x", 1<<10) +
		"\n"
	tpl := `#@ load("@ytt:data", "data")
#@ load("@ytt:yaml", "yaml")
name: #@ yaml.decode(data.read("table.txt"))["name"]
`
	library := workspace.NewRootLibrary([]*files.File{
		files.MustNewFileFromSource(
			files.NewBytesSource("tpl.yml", []byte(tpl))),
		files.MustNewFileFromSource(
			files.NewBytesSource("table.txt", []byte(table))),
	})
	ctx := workspace.LibraryExecutionContext{Current: library, Root: library}

	result, err := factory.New(ctx).Eval(datavalues.NewEmptyEnvelope(), nil,
		nil)
	require.NoError(t, err)
	// The data file is also an output.
	outputs := map[string]string{}
	for _, file := range result.Files {
		outputs[file.RelativePath()] = string(file.Bytes())
	}
	require.Equal(t, "name: "+name+"\n", outputs["tpl.yml"])
}
