/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package extgatling

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// zipDir writes the contents of dir into a newly created zip archive at dst, with
// paths relative to dir. dst must not be inside dir. Anything that is not a
// regular file or a directory is skipped.
func zipDir(dir, dst string) (err error) {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	w := zip.NewWriter(out)
	defer func() {
		if closeErr := w.Close(); err == nil {
			err = closeErr
		}
	}()

	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		name = filepath.ToSlash(name)

		if entry.IsDir() {
			_, err := w.Create(name + "/")
			return err
		}
		if !entry.Type().IsRegular() {
			log.Warn().Msgf("Not adding %s to %s: not a regular file", path, dst)
			return nil
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = in.Close() }()

		target, err := w.Create(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(target, in)
		return err
	})
}

// unzip extracts the zip archive at src into dst and returns the names of the
// entries it did not extract, so the caller can decide whether that is a
// problem. Entries that would escape dst are rejected outright; entries that are
// neither a regular file nor a directory are the skipped ones. Extracted files
// get mode 0644, directories 0755 -- the archive's own mode bits are ignored, as
// they are frequently missing or meaningless depending on the system the archive
// was built on.
func unzip(src, dst string) (skipped []string, err error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		path, err := resolveWithin(dst, f.Name)
		if err != nil {
			return nil, err
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0755); err != nil {
				return nil, err
			}
			continue
		}
		if !f.FileInfo().Mode().IsRegular() {
			skipped = append(skipped, f.Name)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, err
		}
		if err := copyOutOfArchive(f, path); err != nil {
			return nil, err
		}
	}

	return skipped, nil
}

func copyOutOfArchive(f *zip.File, path string) error {
	in, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// resolveWithin joins name onto dir, refusing names that would end up outside of
// it (zip slip).
func resolveWithin(dir, name string) (string, error) {
	local := filepath.FromSlash(name)
	if !filepath.IsLocal(local) {
		return "", fmt.Errorf("archive entry %q would be extracted outside of %s", name, dir)
	}
	return filepath.Join(dir, local), nil
}
