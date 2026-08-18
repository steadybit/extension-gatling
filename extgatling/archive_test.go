/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package extgatling

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_zipDir_stores_the_tree_with_relative_paths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "simulation.log"), "run")
	writeFile(t, filepath.Join(dir, "js", "app.js"), "console.log(1)")
	archive := filepath.Join(t.TempDir(), "report.zip")

	require.NoError(t, zipDir(dir, archive))

	assert.Equal(t, map[string]string{
		"simulation.log": "run",
		"js/":            "",
		"js/app.js":      "console.log(1)",
	}, readArchive(t, archive))
}

func Test_zipDir_truncates_an_existing_archive(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "report.zip")

	first := t.TempDir()
	writeFile(t, filepath.Join(first, "first.log"), "1")
	require.NoError(t, zipDir(first, archive))

	second := t.TempDir()
	writeFile(t, filepath.Join(second, "second.log"), "2")
	require.NoError(t, zipDir(second, archive))

	assert.Equal(t, map[string]string{"second.log": "2"}, readArchive(t, archive),
		"a second run must not append to the archive of the first")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func readArchive(t *testing.T, path string) map[string]string {
	t.Helper()
	r, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	entries := make(map[string]string, len(r.File))
	for _, f := range r.File {
		in, err := f.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(in)
		require.NoError(t, err)
		require.NoError(t, in.Close())
		entries[f.Name] = string(content)
	}
	return entries
}
