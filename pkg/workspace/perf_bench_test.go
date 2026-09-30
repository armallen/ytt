// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package workspace_test

// Benchmarks for embedding ytt: a program that evaluates many templates in one
// process with one LibraryExecutionFactory, the way config generators such as
// cfgen render one template per component. Every input is generated here.
//
//	go test ./pkg/workspace -run '^$' -bench . -benchmem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"carvel.dev/ytt/pkg/cmd/ui"
	"carvel.dev/ytt/pkg/files"
	"carvel.dev/ytt/pkg/workspace"
	"carvel.dev/ytt/pkg/workspace/datavalues"
	"carvel.dev/ytt/pkg/yamlmeta"
)

// BenchmarkEvalSharedFactory evaluates the same template (with a Starlark
// library and a YAML library) repeatedly with one factory; only the data values
// change between evaluations.
func BenchmarkEvalSharedFactory(b *testing.B) {
	dir := writeTree(b, map[string]string{
		"lib/helpers.star":      helpersStar(150),
		"lib/fragments.lib.yml": fragmentsLibYAML,
		"template.yml": `#@ load("@ytt:data", "data")
#@ load("lib/helpers.star", "f1")
#@ load("lib/fragments.lib.yml", "item")
name: #@ data.values.name
items:
#@ for i in range(50):
- #@ item(i, f1(data.values.id))
#@ end
`,
	})
	benchEval(b, dir, "template.yml")
}

// BenchmarkEvalManyLoads imports one symbol per load() statement, a common
// style, from a library that itself loads another library.
func BenchmarkEvalManyLoads(b *testing.B) {
	var tpl strings.Builder
	tpl.WriteString("#@ load(\"@ytt:data\", \"data\")\n")
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&tpl, "#@ load(\"lib/helpers.star\", \"f%d\")\n", i)
	}
	tpl.WriteString("values:\n")
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&tpl, "- #@ f%d(data.values.id)\n", i)
	}
	dir := writeTree(b, map[string]string{
		"lib/common.star":  helpersStar(100),
		"lib/helpers.star": "load(\"common.star\", common_f0=\"f0\")\n" + helpersStar(150) + "def g(x):\n  return common_f0(x)\nend\n",
		"template.yml":     tpl.String(),
	})
	benchEval(b, dir, "template.yml")
}

// BenchmarkDataReadManyFiles reads records with data.read() from a directory
// holding thousands of files.
func BenchmarkDataReadManyFiles(b *testing.B) {
	tree := map[string]string{
		"template.yml": `#@ load("@ytt:data", "data")
#@ load("@ytt:yaml", "yaml")
records:
#@ for i in range(50):
- #@ yaml.decode(data.read("records/r" + str((i * 97 + data.values.id) % 5000) + ".yml"))
#@ end
`,
	}
	for i := 0; i < 5000; i++ {
		tree[fmt.Sprintf("records/r%d.yml", i)] = fmt.Sprintf("id: %d\nname: record-%d\nposition: [%d, %d, %d]\n", i, i, i, i*2, i*3)
	}
	benchEval(b, writeTree(b, tree), "template.yml")
}

// BenchmarkYAMLDecodeLargeFile decodes the same ~0.5 MB data file in every
// evaluation, as templates do when they look up entries in a shared table.
func BenchmarkYAMLDecodeLargeFile(b *testing.B) {
	var table strings.Builder
	for i := 0; i < 5000; i++ {
		fmt.Fprintf(&table, "region-%d:\n  id: %d\n  bounds: [%d, %d, %d, %d]\n  tags: [a, b, c]\n", i, i, i, i+1, i+2, i+3)
	}
	dir := writeTree(b, map[string]string{
		"table.yml": table.String(),
		"template.yml": `#@ load("@ytt:data", "data")
#@ load("@ytt:yaml", "yaml")
#@ table = yaml.decode(data.read("table.yml"))
region: #@ table["region-" + str(data.values.id % 5000)]
`,
	})
	benchEval(b, dir, "template.yml")
}

// BenchmarkFilesFromRelativePath lists a directory tree given as a relative
// path, as the ytt CLI does for -f arguments.
func BenchmarkFilesFromRelativePath(b *testing.B) {
	tree := map[string]string{}
	for i := 0; i < 3000; i++ {
		tree[fmt.Sprintf("d%d/f%d.yml", i%30, i)] = "a: 1\n"
	}
	dir := writeTree(b, tree)
	b.Chdir(filepath.Dir(dir))
	rel := filepath.Base(dir)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := files.NewSortedFilesFromPaths([]string{rel}, files.SymlinkAllowOpts{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOverlayLargeMap applies a small overlay to a document with a large map.
func BenchmarkOverlayLargeMap(b *testing.B) {
	var tpl strings.Builder
	tpl.WriteString("#@ load(\"@ytt:data\", \"data\")\nconfig:\n  id: #@ data.values.id\n")
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&tpl, "  key%d:\n    value: %d\n    list: [1, 2, 3]\n", i, i)
	}
	dir := writeTree(b, map[string]string{
		"template.yml": tpl.String(),
		"overlay.yml": `#@ load("@ytt:overlay", "overlay")
#@overlay/match by=overlay.all
---
config:
  key10:
    value: 99
  #@overlay/match missing_ok=True
  extra: true
`,
	})
	benchEvalOutputs(b, dir, "template.yml", "overlay.yml")
}

func benchEval(b *testing.B, dir, template string) { benchEvalOutputs(b, dir, template) }

// benchEvalOutputs evaluates the library in dir once per iteration with a new
// file list and data values, like an embedder rendering one component at a
// time, sharing one factory across iterations.
func benchEvalOutputs(b *testing.B, dir string, outputs ...string) {
	factory := workspace.NewLibraryExecutionFactory(ui.NewTTY(false), workspace.TemplateLoaderOpts{}, true)
	evalOnce(b, factory, dir, 0, outputs) // fail early, outside the timer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evalOnce(b, factory, dir, i, outputs)
	}
}

func evalOnce(b *testing.B, factory *workspace.LibraryExecutionFactory, dir string, id int, outputs []string) {
	srcs, err := files.NewSortedFilesFromPaths([]string{dir}, files.SymlinkAllowOpts{})
	if err != nil {
		b.Fatal(err)
	}
	for _, f := range srcs {
		out := false
		for _, o := range outputs {
			out = out || f.RelativePath() == o
		}
		f.MarkForOutput(out)
	}
	docSet, err := yamlmeta.NewDocumentSetFromBytes([]byte(fmt.Sprintf("id: %d\nname: component-%d\n", id, id)), yamlmeta.DocSetOpts{})
	if err != nil {
		b.Fatal(err)
	}
	values, err := datavalues.NewEnvelope(docSet.Items[0])
	if err != nil {
		b.Fatal(err)
	}
	lib := workspace.NewRootLibrary(srcs)
	if _, err := factory.New(workspace.LibraryExecutionContext{Current: lib, Root: lib}).Eval(values, nil, nil); err != nil {
		b.Fatal(err)
	}
}

func writeTree(b *testing.B, tree map[string]string) string {
	dir := filepath.Join(b.TempDir(), "lib-root")
	for name, content := range tree {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	return dir
}

// helpersStar returns a Starlark library defining f0..f(n-1) plus some
// module-level constants.
func helpersStar(n int) string {
	var s strings.Builder
	s.WriteString("TABLE = {\n")
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&s, "  \"k%d\": [%d, \"v%d\", %d.5],\n", i, i, i, i)
	}
	s.WriteString("}\n\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&s, "def f%d(x):\n  return {\"id\": x + %d, \"name\": \"f%d-\" + str(x), \"k\": TABLE[\"k%d\"]}\nend\n\n", i, i, i, i%50)
	}
	return s.String()
}

const fragmentsLibYAML = `#@ def item(i, extra):
index: #@ i
extra: #@ extra
labels:
  app: bench
  tier: #@ "t" + str(i % 3)
#@ end
`
