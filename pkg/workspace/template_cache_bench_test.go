// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package workspace_test

// Benchmarks for programs that evaluate the same library many times with one
// LibraryExecutionFactory (e.g. one evaluation per output, with different
// data values). Each benchmark runs with and without a TemplateCache. Every
// input is generated here.
//
//	go test ./pkg/workspace -run '^$' -bench . -benchmem

import (
	"fmt"
	"strings"
	"testing"

	"carvel.dev/ytt/pkg/workspace"
)

const (
	// benchHelpers is the number of functions in the Starlark libraries.
	benchHelpers = 150
	// benchLoads is the number of load() statements in BenchmarkEvalManyLoads.
	benchLoads = 40
	// benchTableSize is the number of entries in each library's TABLE.
	benchTableSize = 50
)

// BenchmarkEvalSharedFactory evaluates the same template (with a Starlark
// library and a YAML library) repeatedly with one factory; only the data
// values change between evaluations.
func BenchmarkEvalSharedFactory(b *testing.B) {
	benchEval(b, writeTree(b, map[string]string{
		"lib/helpers.star":      helpersStar(benchHelpers),
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
	}))
}

// BenchmarkEvalManyLoads imports one symbol per load() statement, a common
// style, from a library that itself loads another library.
func BenchmarkEvalManyLoads(b *testing.B) {
	loads := make([]string, benchLoads)
	values := make([]string, benchLoads)
	for i := range benchLoads {
		loads[i] = fmt.Sprintf("#@ load(\"lib/helpers.star\", \"f%d\")\n", i)
		values[i] = fmt.Sprintf("- #@ f%d(data.values.id)\n", i)
	}
	tpl := "#@ load(\"@ytt:data\", \"data\")\n" + concat(loads) +
		"values:\n" + concat(values)

	benchEval(b, writeTree(b, map[string]string{
		"lib/common.star": helpersStar(benchHelpers),
		"lib/helpers.star": "load(\"common.star\", common_f0=\"f0\")\n" +
			helpersStar(benchHelpers) +
			"def g(x):\n  return common_f0(x)\nend\n",
		"template.yml": tpl,
	}))
}

// BenchmarkEvalLibraryInstances evaluates one library 50 times with
// different data values from a single template, a common way to generate one
// set of resources per entry in a list.
func BenchmarkEvalLibraryInstances(b *testing.B) {
	benchEval(b, writeTree(b, map[string]string{
		"_ytt_lib/app/values.yml":   libraryValuesYAML,
		"_ytt_lib/app/helpers.star": helpersStar(benchHelpers),
		"_ytt_lib/app/deployment.yml": `#@ load("@ytt:data", "data")
#@ load("helpers.star", "f1")
kind: Deployment
metadata:
  name: #@ data.values.name
  labels: #@ {"v": f1(data.values.port)["name"]}
spec:
  containers:
  #@ for i in range(5):
  - name: #@ data.values.name + "-" + str(i)
    port: #@ data.values.port + i
  #@ end
`,
		"template.yml": `#@ load("@ytt:library", "library")
#@ load("@ytt:template", "template")
#@ app = library.get("app")
#@ for i in range(50):
#@   values = {"name": "app" + str(i), "port": 8000 + i}
--- #@ template.replace(app.with_data_values(values).eval())
#@ end
`,
	}))
}

// benchEval evaluates the library in dir once per iteration, with a new file
// list and data values, sharing one factory across iterations, without and
// with a template cache.
func benchEval(b *testing.B, dir string) {
	for _, bc := range []struct {
		name  string
		cache *workspace.TemplateCache
	}{
		{"no-cache", nil},
		{"cache", workspace.NewTemplateCache()},
	} {
		b.Run(bc.name, func(b *testing.B) {
			benchEvalWith(b, newFactory(bc.cache), dir)
		})
	}
}

func benchEvalWith(b *testing.B, factory *workspace.LibraryExecutionFactory,
	dir string,
) {
	// Fail early, outside the timer.
	if _, err := evalLibrary(factory, dir, 0); err != nil {
		b.Fatal(err)
	}
	id := 0
	for b.Loop() {
		if _, err := evalLibrary(factory, dir, id); err != nil {
			b.Fatal(err)
		}
		id++
	}
}

// helpersStar returns a Starlark library defining f0..f(n-1) plus a
// module-level table.
func helpersStar(n int) string {
	table := make([]string, benchTableSize)
	for i := range table {
		table[i] = fmt.Sprintf("  \"k%d\": [%d, \"v%d\", %d.5],\n", i, i, i, i)
	}
	funcs := make([]string, n)
	for i := range funcs {
		funcs[i] = fmt.Sprintf("def f%d(x):\n"+
			"  return {\"id\": x + %d, \"name\": \"f%d-\" + str(x), "+
			"\"k\": TABLE[\"k%d\"]}\nend\n\n", i, i, i, i%benchTableSize)
	}
	return "TABLE = {\n" + concat(table) + "}\n\n" + concat(funcs)
}

func concat(lines []string) string { return strings.Join(lines, "") }

const libraryValuesYAML = `#@data/values
---
name: ""
port: 0
`

const fragmentsLibYAML = `#@ def item(i, extra):
index: #@ i
extra: #@ extra
labels:
  app: bench
  tier: #@ "t" + str(i % 3)
#@ end
`
