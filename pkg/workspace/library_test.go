// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package workspace_test

import (
	"testing"

	"carvel.dev/ytt/pkg/files"
	"carvel.dev/ytt/pkg/workspace"
	"github.com/stretchr/testify/require"
)

func TestFindFile(t *testing.T) {
	newFile := func(path, content string) *files.File {
		return files.MustNewFileFromSource(
			files.NewBytesSource(path, []byte(content)))
	}
	root := workspace.NewRootLibrary([]*files.File{
		newFile("a.yml", "root a"),
		newFile("dir/a.yml", "dir a"),
		newFile("dir/b.yml", "first dir b"),
		newFile("dir/b.yml", "second dir b"),
	})

	for path, expected := range map[string]string{
		"a.yml":     "root a",
		"dir/a.yml": "dir a",
		"dir/b.yml": "first dir b",
	} {
		found, err := root.FindFile(path)
		require.NoError(t, err)
		bs, err := found.File.Bytes()
		require.NoError(t, err)
		require.Equal(t, expected, string(bs), "finding %s", path)
	}

	for _, path := range []string{"b.yml", "dir/c.yml", "other/a.yml"} {
		_, err := root.FindFile(path)
		require.Error(t, err, "finding %s", path)
	}
}
