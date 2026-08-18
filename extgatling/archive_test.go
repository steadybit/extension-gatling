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

func Test_unzip_extracts_the_tree(t *testing.T) {
	archive := writeArchive(t, map[string]string{
		"BasicSimulation.scala": "class BasicSimulation",
		"data/users.csv":        "id\n1",
	})
	dst := t.TempDir()

	skipped, err := unzip(archive, dst)

	require.NoError(t, err)
	assert.Empty(t, skipped)
	assert.Equal(t, "class BasicSimulation", readFile(t, filepath.Join(dst, "BasicSimulation.scala")))
	assert.Equal(t, "id\n1", readFile(t, filepath.Join(dst, "data", "users.csv")))
}

func Test_unzip_rejects_entries_escaping_the_destination(t *testing.T) {
	archive := writeArchive(t, map[string]string{"../escaped.scala": "nope"})
	dst := filepath.Join(t.TempDir(), "src")
	require.NoError(t, os.Mkdir(dst, 0755))

	_, err := unzip(archive, dst)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "would be extracted outside of")
	assert.NoFileExists(t, filepath.Join(filepath.Dir(dst), "escaped.scala"))
}

func Test_zipDir_and_unzip_round_trip(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "nested", "deep", "file.txt"), "content")
	archive := filepath.Join(t.TempDir(), "report.zip")
	require.NoError(t, zipDir(src, archive))

	dst := t.TempDir()
	skipped, err := unzip(archive, dst)

	require.NoError(t, err)
	assert.Empty(t, skipped)
	assert.Equal(t, "content", readFile(t, filepath.Join(dst, "nested", "deep", "file.txt")))
}

func Test_unzip_reports_entries_it_did_not_extract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sources.zip")
	out, err := os.Create(path)
	require.NoError(t, err)
	w := zip.NewWriter(out)
	symlink := &zip.FileHeader{Name: "link.scala", Method: zip.Deflate}
	symlink.SetMode(os.ModeSymlink | 0777)
	entry, err := w.CreateHeader(symlink)
	require.NoError(t, err)
	_, err = entry.Write([]byte("BasicSimulation.scala"))
	require.NoError(t, err)
	entry, err = w.Create("BasicSimulation.scala")
	require.NoError(t, err)
	_, err = entry.Write([]byte("class BasicSimulation"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, out.Close())
	dst := t.TempDir()

	skipped, err := unzip(path, dst)

	require.NoError(t, err)
	assert.Equal(t, []string{"link.scala"}, skipped)
	assert.NoFileExists(t, filepath.Join(dst, "link.scala"))
	assert.Equal(t, "class BasicSimulation", readFile(t, filepath.Join(dst, "BasicSimulation.scala")),
		"the entries around a skipped one must still be extracted")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

// writeArchive builds a zip archive from name -> content, using the names
// verbatim so tests can include entries a well-behaved archiver would not emit.
func writeArchive(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sources.zip")
	out, err := os.Create(path)
	require.NoError(t, err)
	defer func() { _ = out.Close() }()

	w := zip.NewWriter(out)
	for name, content := range entries {
		entry, err := w.Create(name)
		require.NoError(t, err)
		_, err = entry.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return path
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
