// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package template_test

import (
	"sync"
	"testing"

	"carvel.dev/ytt/pkg/template"
)

func TestNewInstructionSetConcurrentNamesAreUnique(t *testing.T) {
	const n = 50
	names := make([]string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			names[i] = template.NewInstructionSet().StartNode.Name
		}(i)
	}
	wg.Wait()

	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			t.Fatalf("expected unique instruction names, got %q twice", name)
		}
		seen[name] = true
	}
}
