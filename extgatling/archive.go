/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package extgatling

import (
	"archive/zip"
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
