// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package template_test

import (
	"testing"

	cmdtpl "carvel.dev/ytt/pkg/cmd/template"
	"carvel.dev/ytt/pkg/cmd/ui"
	"carvel.dev/ytt/pkg/files"
	"github.com/stretchr/testify/require"
)

// A library module loaded by several load() statements is evaluated once per
// template evaluation, so every load() gets the same values. Starlark
// functions compare equal only to themselves.
func TestLoadEvaluatesEachModuleOnce(t *testing.T) {
	tests := []struct {
		name    string
		library string
		data    string
	}{
		{
			name:    "starlark",
			library: "funcs.star",
			data:    "def f():\n  return 1\nend\n",
		},
		{
			name:    "yaml",
			library: "funcs.lib.yml",
			data:    "#@ def f():\nkey: value\n#@ end\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tpl := "#@ load(\"" + tc.library + "\", first=\"f\")\n" +
				"#@ load(\"" + tc.library + "\", second=\"f\")\n" +
				"same: #@ first == second\n"
			filesToProcess := []*files.File{
				files.MustNewFileFromSource(
					files.NewBytesSource("tpl.yml", []byte(tpl))),
				files.MustNewFileFromSource(
					files.NewBytesSource(tc.library, []byte(tc.data))),
			}

			out := cmdtpl.NewOptions().RunWithFiles(
				cmdtpl.Input{Files: filesToProcess}, ui.NewTTY(false))
			require.NoError(t, out.Err)
			require.Len(t, out.Files, 1, "unexpected number of output files")
			require.Equal(t, "same: true\n", string(out.Files[0].Bytes()))
		})
	}
}
