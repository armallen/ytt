// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package workspace

import (
	"sync"

	"carvel.dev/ytt/pkg/cmd/ui"
	"carvel.dev/ytt/pkg/template"
	"github.com/k14s/starlark-go/starlark"
)

// progCacheEntry holds a compiled Starlark program together with the InstructionSet
// whose names are baked into the program as predeclared globals. Both must be
// applied together when reusing a cached program.
type progCacheEntry struct {
	prog         *starlark.Program
	instructions *template.InstructionSet
}

// LibraryExecutionContext holds the total set of inputs that are involved in a LibraryExecution.
type LibraryExecutionContext struct {
	Current *Library // the target library that will be executed/evaluated.
	Root    *Library // reference to the root library to support accessing "absolute path" loading of files.
}

// LibraryExecutionFactory holds configuration for and produces instances of LibraryExecution's.
type LibraryExecutionFactory struct {
	ui                 ui.UI
	templateLoaderOpts TemplateLoaderOpts

	skipDataValuesValidation bool

	// progCacheMu and progCache are shared across derived factories so that compiled Starlark
	// programs are reused across all components processed within a single cfgen run.
	progCacheMu *sync.RWMutex
	progCache   map[string]*progCacheEntry
}

// NewLibraryExecutionFactory configures a new instance of a LibraryExecutionFactory.
func NewLibraryExecutionFactory(ui ui.UI, templateLoaderOpts TemplateLoaderOpts, skipDataValuesValidation bool) *LibraryExecutionFactory {
	return &LibraryExecutionFactory{
		ui:                       ui,
		templateLoaderOpts:       templateLoaderOpts,
		skipDataValuesValidation: skipDataValuesValidation,
		progCacheMu:              &sync.RWMutex{},
		progCache:                map[string]*progCacheEntry{},
	}
}

func (f *LibraryExecutionFactory) getCachedProgram(path string) (*progCacheEntry, bool) {
	f.progCacheMu.RLock()
	defer f.progCacheMu.RUnlock()
	entry, ok := f.progCache[path]
	return entry, ok
}

func (f *LibraryExecutionFactory) setCachedProgram(path string, prog *starlark.Program, instructions *template.InstructionSet) {
	f.progCacheMu.Lock()
	defer f.progCacheMu.Unlock()
	f.progCache[path] = &progCacheEntry{prog: prog, instructions: instructions}
}

// WithTemplateLoaderOptsOverrides produces a new LibraryExecutionFactory identical to this one, except it configures
// its TemplateLoader with the merge of the supplied TemplateLoaderOpts over this factory's configuration.
func (f *LibraryExecutionFactory) WithTemplateLoaderOptsOverrides(overrides TemplateLoaderOptsOverrides) *LibraryExecutionFactory {
	newF := NewLibraryExecutionFactory(f.ui, f.templateLoaderOpts.Merge(overrides), f.skipDataValuesValidation)
	newF.progCacheMu = f.progCacheMu
	newF.progCache = f.progCache
	return newF
}

// ThatSkipsDataValuesValidations produces a new LibraryExecutionFactory identical to this one, except it might also
// skip running validation rules over Data Values.
//
// If a LibraryExecutionFactory has already been configured to skip validations, calling this method with `true` has
// no effect. This stems from the assumption that the downstream user is the most informed whether validations ought to
// be run.
func (f *LibraryExecutionFactory) ThatSkipsDataValuesValidations(skipDataValuesValidation bool) *LibraryExecutionFactory {
	newF := NewLibraryExecutionFactory(f.ui, f.templateLoaderOpts, f.skipDataValuesValidation || skipDataValuesValidation)
	newF.progCacheMu = f.progCacheMu
	newF.progCache = f.progCache
	return newF
}

// New produces a new instance of a LibraryExecution, set with the configuration and dependencies of this factory.
func (f *LibraryExecutionFactory) New(ctx LibraryExecutionContext) *LibraryExecution {
	return NewLibraryExecution(ctx, f.ui, f.templateLoaderOpts, f, f.skipDataValuesValidation)
}
