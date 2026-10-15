// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package workspace

import (
	"sync"

	"carvel.dev/ytt/pkg/files"
	"carvel.dev/ytt/pkg/template"
)

// TemplateCache keeps compiled template files, so that evaluations sharing it
// skip parsing and compiling files they have already seen. Data values,
// overlays and other evaluation state are not cached.
//
// Entries are keyed by file description (e.g. its path), not by content:
// files must not change while a TemplateCache is in use. Create a new
// TemplateCache when they do. Entries are never evicted.
//
// A TemplateCache is safe for concurrent use. Attach one to a factory with
// LibraryExecutionFactory.WithTemplateCache.
type TemplateCache struct {
	mu        sync.RWMutex
	templates map[templateKey]*template.CompiledTemplate
}

// NewTemplateCache returns an empty TemplateCache.
func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		templates: map[templateKey]*template.CompiledTemplate{},
	}
}

// templateKind is the kind of template a file is compiled as.
type templateKind string

const (
	starlarkTemplate templateKind = "starlark"
)

// templateKey identifies a file and everything that changes how it compiles.
type templateKey struct {
	kind        templateKind
	description string
	isTemplate  bool
	isLibrary   bool
	opts        TemplateLoaderOpts
}

func newTemplateKey(
	kind templateKind, file *files.File, opts TemplateLoaderOpts,
) templateKey {
	return templateKey{
		kind:        kind,
		description: file.Description(),
		isTemplate:  file.IsTemplate(),
		isLibrary:   file.IsLibrary(),
		opts:        opts,
	}
}

// The methods below treat a nil *TemplateCache as a cache that is always
// empty and ignores writes, which is how a factory without a cache behaves.

// lookupTemplate returns a copy of the template cached under key, to evaluate.
func (c *TemplateCache) lookupTemplate(
	key templateKey,
) (*template.CompiledTemplate, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	tpl, ok := c.templates[key]
	if !ok {
		return nil, false
	}
	return tpl.CloneForEval(), true
}

// storeTemplate caches a copy of tpl, which has been evaluated, so that the
// copy holds its compiled program.
func (c *TemplateCache) storeTemplate(
	key templateKey, tpl *template.CompiledTemplate,
) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.templates[key] = tpl.CloneForEval()
}
