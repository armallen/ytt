// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package workspace_test

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"carvel.dev/ytt/pkg/cmd/ui"
	"carvel.dev/ytt/pkg/files"
	"carvel.dev/ytt/pkg/workspace"
	"carvel.dev/ytt/pkg/workspace/datavalues"
	"carvel.dev/ytt/pkg/yamlmeta"
)

const (
	// cacheTestIDs is the number of data values each test evaluates with.
	cacheTestIDs = 4
	// cacheTestWorkers is the number of concurrent evaluations.
	cacheTestWorkers = 20
	// cacheTestRepeats is the number of cached evaluations per data value.
	cacheTestRepeats = 2
)

// cacheTestLibrary has every kind of file: a YAML template, a YAML library,
// a Starlark library, plain YAML and a text template.
var cacheTestLibrary = map[string]string{
	"lib/helpers.star": `def double(x):
  return 2 * x
end
`,
	"lib/fragments.lib.yml": `#@ def item(i):
index: #@ i
#@ end
`,
	"template.yml": `#@ load("@ytt:data", "data")
#@ load("lib/helpers.star", "double")
#@ load("lib/fragments.lib.yml", "item")
id: #@ double(data.values.id)
items:
#@ for i in range(data.values.id % 3):
- #@ item(i)
#@ end
`,
	"plain.yml": "plain: true\n",
	"template.txt": `(@ load("@ytt:data", "data") @)` +
		`id=(@= str(data.values.id) @)` + "\n",
}

func TestTemplateCacheIsOffByDefault(t *testing.T) {
	factory := newFactory(nil)

	// Same path, different content: without a cache, each evaluation sees its
	// own files. "#@ x = 1" makes the file a template, which is what a cache
	// would keep.
	for _, want := range []string{"a: 1\n", "a: 2\n"} {
		tree := map[string]string{"template.yml": "#@ x = 1\n" + want}
		dir := writeTree(t, tree)
		if got := evalDir(t, factory, dir, 0); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	}
}

func TestTemplateCacheMatchesUncachedEvaluation(t *testing.T) {
	uncached := newFactory(nil)
	cached := newFactory(workspace.NewTemplateCache())
	dir := writeTree(t, cacheTestLibrary)

	for id := range cacheTestIDs {
		want := evalDir(t, uncached, dir, id)
		for range cacheTestRepeats {
			if got := evalDir(t, cached, dir, id); got != want {
				t.Fatalf("id %d: expected:\n%s\ngot:\n%s", id, want, got)
			}
		}
	}
}

func TestTemplateCacheConcurrentEvaluations(t *testing.T) {
	uncached := newFactory(nil)
	cached := newFactory(workspace.NewTemplateCache())
	dir := writeTree(t, cacheTestLibrary)

	want := make([]string, cacheTestWorkers)
	for id := range want {
		want[id] = evalDir(t, uncached, dir, id)
	}

	got := make([]string, cacheTestWorkers)
	var wg sync.WaitGroup
	for id := range got {
		wg.Go(func() { got[id] = evalDir(t, cached, dir, id) })
	}
	wg.Wait()

	for id := range want {
		if got[id] != want[id] {
			t.Fatalf("id %d: expected:\n%s\ngot:\n%s", id, want[id], got[id])
		}
	}
}

// Files with the same relative path under different directories are
// different files to the cache.
func TestTemplateCacheKeysByFileDescription(t *testing.T) {
	cached := newFactory(workspace.NewTemplateCache())

	for _, factor := range []string{"2", "3"} {
		tree := maps.Clone(cacheTestLibrary)
		tree["lib/helpers.star"] = fmt.Sprintf(
			"def double(x):\n  return %s * x\nend\n", factor)
		dir := writeTree(t, tree)

		want := evalDir(t, newFactory(nil), dir, 1)
		if got := evalDir(t, cached, dir, 1); got != want {
			t.Fatalf("factor %s: expected:\n%s\ngot:\n%s", factor, want, got)
		}
	}
}

func newFactory(
	cache *workspace.TemplateCache,
) *workspace.LibraryExecutionFactory {
	return workspace.NewLibraryExecutionFactory(
		ui.NewTTY(false), workspace.TemplateLoaderOpts{}, true).
		WithTemplateCache(cache)
}

// writeTree writes tree, a map of relative paths to contents, under a new
// temporary directory, and returns that directory.
func writeTree(tb testing.TB, tree map[string]string) string {
	const dirPerm, filePerm = 0o755, 0o644

	dir := filepath.Join(tb.TempDir(), "lib-root")
	for name, content := range tree {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), filePerm); err != nil {
			tb.Fatal(err)
		}
	}
	return dir
}

// evalDir evaluates the library in dir with data value id, and returns its
// YAML output followed by its other output files.
func evalDir(tb testing.TB, factory *workspace.LibraryExecutionFactory,
	dir string, id int,
) string {
	result, err := evalLibrary(factory, dir, id)
	if err != nil {
		tb.Error(err) // not Fatal: may run in a goroutine
		return ""
	}
	out, err := result.DocSet.AsBytes()
	if err != nil {
		tb.Error(err)
		return ""
	}
	for _, f := range result.Files {
		if f.Type() != files.TypeYAML {
			out = append(out, f.Bytes()...)
		}
	}
	return string(out)
}

// evalLibrary evaluates the library in dir with data values id and name.
func evalLibrary(factory *workspace.LibraryExecutionFactory, dir string,
	id int,
) (*workspace.EvalResult, error) {
	srcs, err := files.NewSortedFilesFromPaths(
		[]string{dir}, files.SymlinkAllowOpts{})
	if err != nil {
		return nil, err
	}
	docSet, err := yamlmeta.NewDocumentSetFromBytes(
		fmt.Appendf(nil, "id: %d\nname: component-%d\n", id, id),
		yamlmeta.DocSetOpts{})
	if err != nil {
		return nil, err
	}
	values, err := datavalues.NewEnvelope(docSet.Items[0])
	if err != nil {
		return nil, err
	}
	lib := workspace.NewRootLibrary(srcs)
	ctx := workspace.LibraryExecutionContext{Current: lib, Root: lib}
	return factory.New(ctx).Eval(values, nil, nil)
}
