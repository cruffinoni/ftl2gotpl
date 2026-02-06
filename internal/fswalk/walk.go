package fswalk

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// TemplateFile stores absolute and root-relative paths for one template.
type TemplateFile struct {
	AbsPath string
	RelPath string
}

// normalizePattern returns a usable glob and defaults to **/*.ftl.
func normalizePattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "**/*.ftl"
	}
	return filepath.ToSlash(pattern)
}

// DiscoverTemplates finds files under root matching the glob pattern.
func DiscoverTemplates(root string, pattern string) ([]TemplateFile, error) {
	root = filepath.Clean(root)
	matcher := normalizePattern(pattern)

	var files []TemplateFile
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("compute relative path for %q: %w", path, err)
		}

		matched, err := doublestar.PathMatch(matcher, filepath.ToSlash(relPath))
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if !matched {
			return nil
		}

		files = append(files, TemplateFile{
			AbsPath: path,
			RelPath: relPath,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})

	return files, nil
}

// MirrorOutputPath maps a relative input path to an output path and extension.
func MirrorOutputPath(outRoot string, relPath string, ext string) string {
	cleanRel := filepath.Clean(relPath)
	if ext != "" {
		oldExt := filepath.Ext(cleanRel)
		cleanRel = strings.TrimSuffix(cleanRel, oldExt) + ext
	}
	return filepath.Join(outRoot, cleanRel)
}

// EnsureParentDir creates the parent directory tree for a target file path.
func EnsureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

// CopyFile copies a file while creating parent directories for destination.
func CopyFile(srcPath string, dstPath string) error {
	if err := EnsureParentDir(dstPath); err != nil {
		return err
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}
