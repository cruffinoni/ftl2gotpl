package fswalk

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestDiscoverTemplates(t *testing.T) {
	root := t.TempDir()

	mustWrite(t, filepath.Join(root, "a.ftl"), "a")
	mustWrite(t, filepath.Join(root, "nested", "b.ftl"), "b")
	mustWrite(t, filepath.Join(root, "nested", "c.txt"), "c")

	got, err := DiscoverTemplates(root, "**/*.ftl")
	require.NoError(t, err)

	var rel []string
	for _, f := range got {
		rel = append(rel, filepath.ToSlash(f.RelPath))
	}

	want := []string{"a.ftl", "nested/b.ftl"}
	require.True(t, slices.Equal(rel, want))
}

func TestMirrorOutputPath(t *testing.T) {
	got := filepath.ToSlash(MirrorOutputPath("out", "foo/bar/a.ftl", ".gotmpl"))
	want := "out/foo/bar/a.gotmpl"
	require.Equal(t, want, got)
}
