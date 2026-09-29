// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package workspace

import (
	"carvel.dev/ytt/pkg/cmd/ui"
	"carvel.dev/ytt/pkg/yttlibrary"
)

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

	// yamlDecodeCache, when set, keeps what yaml.decode parsed across
	// evaluations. Derived factories share it.
	yamlDecodeCache *yttlibrary.YAMLDecodeCache
}

// NewLibraryExecutionFactory configures a new instance of a LibraryExecutionFactory.
func NewLibraryExecutionFactory(ui ui.UI, templateLoaderOpts TemplateLoaderOpts, skipDataValuesValidation bool) *LibraryExecutionFactory {
	return &LibraryExecutionFactory{
		ui:                       ui,
		templateLoaderOpts:       templateLoaderOpts,
		skipDataValuesValidation: skipDataValuesValidation,
	}
}

// WithYAMLDecodeCache produces a new LibraryExecutionFactory identical to
// this one, except yaml.decode in the templates it, or a factory derived from
// it, evaluates keeps parsed values in cache. A nil cache disables caching.
func (f *LibraryExecutionFactory) WithYAMLDecodeCache(
	cache *yttlibrary.YAMLDecodeCache,
) *LibraryExecutionFactory {
	newF := *f
	newF.yamlDecodeCache = cache
	return &newF
}

// WithTemplateLoaderOptsOverrides produces a new LibraryExecutionFactory identical to this one, except it configures
// its TemplateLoader with the merge of the supplied TemplateLoaderOpts over this factory's configuration.
func (f *LibraryExecutionFactory) WithTemplateLoaderOptsOverrides(overrides TemplateLoaderOptsOverrides) *LibraryExecutionFactory {
	newF := *f
	newF.templateLoaderOpts = f.templateLoaderOpts.Merge(overrides)
	return &newF
}

// ThatSkipsDataValuesValidations produces a new LibraryExecutionFactory identical to this one, except it might also
// skip running validation rules over Data Values.
//
// If a LibraryExecutionFactory has already been configured to skip validations, calling this method with `true` has
// no effect. This stems from the assumption that the downstream user is the most informed whether validations ought to
// be run.
func (f *LibraryExecutionFactory) ThatSkipsDataValuesValidations(skipDataValuesValidation bool) *LibraryExecutionFactory {
	newF := *f
	newF.skipDataValuesValidation = f.skipDataValuesValidation ||
		skipDataValuesValidation
	return &newF
}

// New produces a new instance of a LibraryExecution, set with the configuration and dependencies of this factory.
func (f *LibraryExecutionFactory) New(ctx LibraryExecutionContext) *LibraryExecution {
	return NewLibraryExecution(ctx, f.ui, f.templateLoaderOpts, f, f.skipDataValuesValidation)
}
