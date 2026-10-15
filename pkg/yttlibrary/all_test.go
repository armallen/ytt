// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package yttlibrary

import (
	"reflect"
	"testing"

	"carvel.dev/ytt/pkg/cmd/ui"
	"github.com/k14s/starlark-go/starlark"
)

// Modules built on first load are built once, and copies of the API share
// them.
func TestFindModuleBuildsLazyModulesOnce(t *testing.T) {
	api := NewAPI(nil, NewDataModule(starlark.None, nil), nil,
		ui.NewCustomWriterTTY(false, nil, nil))
	apiCopy := api

	for _, name := range []string{"assert", "math", "template", "data"} {
		first := findModule(t, api, name)
		second := findModule(t, apiCopy, name)
		if reflect.ValueOf(first).UnsafePointer() !=
			reflect.ValueOf(second).UnsafePointer() {
			t.Errorf("expected module %s to be built once", name)
		}
	}

	if _, err := api.FindModule("unknown"); err == nil {
		t.Error("expected an error loading an unknown module")
	}
}

func findModule(t *testing.T, api API, name string) starlark.StringDict {
	t.Helper()
	module, err := api.FindModule(name)
	if err != nil {
		t.Fatalf("loading %s: %v", name, err)
	}
	if _, found := module[name]; !found {
		t.Fatalf("expected module %s to define %s, got %v", name, name,
			module.Keys())
	}
	return module
}
