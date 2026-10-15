package main

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// bundleFiles are the documentation files shipped in the docs bundle, relative to the repo root.
func bundleFiles(root string) ([]string, error) {
	files := []string{"README.md", "CHANGELOG.md", "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.md"}
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".md") {
			rel, _ := filepath.Rel(root, p)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// bundle writes a zip of the documentation. It is reproducible: entries are sorted,
// timestamps fixed and permissions normalised, so the same sources give the same bytes.
func bundle(root, out string) error {
	files, err := bundleFiles(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil { //nolint:gosec // dist directory
		return err
	}
	f, err := os.Create(out) //nolint:gosec // output path chosen by the build script
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, rel)) //nolint:gosec // repo-relative
		if err != nil {
			_ = f.Close()
			return err
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: "lumen-docs/" + rel, Method: zip.Deflate, Modified: fixed})
		if err != nil {
			_ = f.Close()
			return err
		}
		if _, err := w.Write(data); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "docgen: wrote %s (%d files)\n", out, len(files))
	return nil
}
