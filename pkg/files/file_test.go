// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package files_test

import (
	"carvel.dev/ytt/pkg/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewSortedFilesFromPaths(t *testing.T) {
	inputPaths := []string{
		"../yamltemplate/filetests/def.tpltest",
		"https://example.org/test.yaml?hello=world&foo=bar",
		"an/overwritten/path/to/file.yaml=https://example.org/test.yaml?hello=world&foo=bar"}
	filesToProcess, err := files.NewSortedFilesFromPaths(inputPaths, files.SymlinkAllowOpts{true, []string{}})

	require.NoError(t, err)

	assert.Equal(t, filesToProcess[0].RelativePath(), "def.tpltest")
	assert.Equal(t, filesToProcess[1].RelativePath(), "test.yaml")
	assert.Equal(t, filesToProcess[1].Description(), "HTTP URL 'https://example.org/test.yaml?hello=world&foo=bar'")
	assert.Equal(t, filesToProcess[2].RelativePath(), "an/overwritten/path/to/file.yaml")
	assert.Equal(t, filesToProcess[2].Description(), "HTTP URL 'https://example.org/test.yaml?hello=world&foo=bar'")
}

func TestFileClone(t *testing.T) {
	orig := files.MustNewFileFromSource(files.NewCachedSource(files.NewBytesSource("dir/tpl.yml", []byte("a: 1"))))
	orig.MarkForOutput(false)
	orig.MarkTemplate(false)
	orig.MarkRelativePath("renamed.yml")

	clone := orig.Clone()

	t.Run("shares the source and path but none of the marks", func(t *testing.T) {
		require.Equal(t, orig.Description(), clone.Description())
		require.Equal(t, "dir/tpl.yml", clone.OriginalRelativePath())
		require.Equal(t, "dir/tpl.yml", clone.RelativePath())
		require.True(t, clone.IsForOutput())
		require.True(t, clone.IsTemplate())

		bs, err := clone.Bytes()
		require.NoError(t, err)
		require.Equal(t, []byte("a: 1"), bs)
	})

	t.Run("marking the clone leaves the original unchanged", func(t *testing.T) {
		clone.MarkRelativePath("other.yml")
		clone.MarkForOutput(true)

		require.Equal(t, "renamed.yml", orig.RelativePath())
		require.False(t, orig.IsForOutput())
		require.False(t, orig.IsTemplate())
	})
}
