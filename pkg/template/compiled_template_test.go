// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package template_test

import (
	"strings"
	"sync"
	"testing"

	"carvel.dev/ytt/pkg/template"
	"github.com/k14s/starlark-go/starlark"
)

func TestEvalExecutesStarlarkAndReturnsGlobals(t *testing.T) {
	data := `
def hello():
  return "hello"
end
a = hello()
`
	instructions := template.NewInstructionSet()
	compiledTemplate := template.NewCompiledTemplate(
		"stdin", template.NewCodeFromBytes([]byte(data), instructions),
		instructions, template.NewNodes(), template.EvaluationCtxDialects{})

	loader := template.NoopCompiledTemplateLoader{}
	thread := &starlark.Thread{Name: "test", Load: loader.Load}

	resultGlobals, resultVal, err := compiledTemplate.Eval(thread, loader)
	if err != nil {
		t.Fatalf("Evaluating starlark template: %s", err)
	}

	if resultGlobals["a"] != starlark.String("hello") {
		t.Fatalf("Expected global to be set, but was %#v", resultGlobals["a"])
	}

	if resultVal != nil {
		t.Fatalf("Expected result val to be nil, but was %#v", resultVal)
	}
}

func TestEvalConcurrentTemplates(t *testing.T) {
	// Uses features gated by Starlark dialect flags: floats, sets, lambdas and
	// global reassignment.
	data := `
a = 1.5
a = sorted(set([3, 1]))
f = lambda x: x * 2
b = f(a[0])
`
	// Instruction sets are created up front: this test is about compiling and
	// evaluating templates concurrently.
	const n = 20
	instructionSets := make([]*template.InstructionSet, n)
	for i := range instructionSets {
		instructionSets[i] = template.NewInstructionSet()
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(instructions *template.InstructionSet) {
			defer wg.Done()
			compiledTemplate := template.NewCompiledTemplate(
				"stdin", template.NewCodeFromBytes([]byte(data), instructions),
				instructions, template.NewNodes(), template.EvaluationCtxDialects{})

			loader := template.NoopCompiledTemplateLoader{}
			thread := &starlark.Thread{Name: "test", Load: loader.Load}

			resultGlobals, _, err := compiledTemplate.Eval(thread, loader)
			if err != nil {
				t.Errorf("Evaluating starlark template: %s", err)
				return
			}
			if resultGlobals["b"] != starlark.MakeInt(2) {
				t.Errorf("Expected b to be 2, but was %#v", resultGlobals["b"])
			}
		}(instructionSets[i])
	}
	wg.Wait()
}

func TestEvalIgnoresIndentationInStarlark(t *testing.T) {
	data := `
		if True:
a = "evaluated"
		end
`
	instructions := template.NewInstructionSet()
	compiledTemplate := template.NewCompiledTemplate(
		"stdin", template.NewCodeFromBytes([]byte(data), instructions),
		instructions, template.NewNodes(), template.EvaluationCtxDialects{})

	loader := template.NewNoopCompiledTemplateLoader(compiledTemplate)
	thread := &starlark.Thread{Name: "test", Load: loader.Load}

	resultGlobals, _, err := compiledTemplate.Eval(thread, loader)
	if err != nil {
		t.Fatalf("Evaluating starlark template: %s", err)
	}

	if resultGlobals["a"] != starlark.String("evaluated") {
		t.Fatalf("Expected variable 'a' to have been set, but was \"%#v\"", resultGlobals["a"])
	}
}
func TestEvalPreservesIndentationOfContinuedLines(t *testing.T) {
	data := `
		if True:
	multiline = "[\
   ]"
		end
`
	instructions := template.NewInstructionSet()
	compiledTemplate := template.NewCompiledTemplate(
		"stdin", template.NewCodeFromBytes([]byte(data), instructions),
		instructions, template.NewNodes(), template.EvaluationCtxDialects{})

	loader := template.NewNoopCompiledTemplateLoader(compiledTemplate)
	thread := &starlark.Thread{Name: "test", Load: loader.Load}

	resultGlobals, _, err := compiledTemplate.Eval(thread, loader)
	if err != nil {
		t.Fatalf("Evaluating starlark template: %s", err)
	}

	if resultGlobals["multiline"] != starlark.String("[   ]") {
		t.Fatalf("Expected multiline to contain the entire multiline string, but was \"%#v\"", resultGlobals["multiline"])
	}
}

// bug test: fixed by bc93b02
func TestEvalReturnsErrorEvenWhenFailsToCompileOnFirstLine(t *testing.T) {
	data := `badsyntax`
	instructions := template.NewInstructionSet()
	compiledTemplate := template.NewCompiledTemplate(
		"stdin", template.NewCodeFromBytes([]byte(data), instructions),
		instructions, template.NewNodes(), template.EvaluationCtxDialects{})

	loader := template.NewNoopCompiledTemplateLoader(compiledTemplate)
	thread := &starlark.Thread{Name: "test", Load: loader.Load}

	_, _, err := compiledTemplate.Eval(thread, loader)
	if err == nil {
		t.Fatalf("Expected eval to return err")
	}

	matchString := "undefined: badsyntax"
	if !strings.Contains(err.Error(), matchString) {
		t.Fatalf("\nExpected:\n%s\n\nto contain:\n\n%s\n", err.Error(), matchString)
	}
}
