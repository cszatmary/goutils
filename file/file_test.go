package file_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TouchBistro/goutils/file"
)

func TestExists(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"dir exists", "testdata/text_tests", true},
		{"file exists", "testdata/text_tests/hype.md", true},
		{"does not exists", "testdata/notafile.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := file.Exists(tt.path)
			if got != tt.want {
				t.Errorf("got %t, want %t", got, tt.want)
			}
		})
	}
}

func TestDownload(t *testing.T) {
	tmpdir := t.TempDir()
	downloadPath := filepath.Join(tmpdir, "builds", "release.build")
	const content = `pretend this is a really important file
	use your imagination`
	r := strings.NewReader(content)
	n, err := file.Download(downloadPath, r)
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	wantN := int64(len(content))
	if n != wantN {
		t.Errorf("got %d bytes written, want %d", n, wantN)
	}

	// Make sure file was actually written
	data, err := ioutil.ReadFile(downloadPath)
	if err != nil {
		t.Fatalf("failed to read file %v", err)
	}
	gotContent := string(data)
	if gotContent != content {
		t.Errorf("got %q, want %q", gotContent, content)
	}
}

func TestCopyFile(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "src")
	dst := filepath.Join(tmpdir, "dst")
	const content = `this is some file content`
	err := ioutil.WriteFile(src, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to write file %v", err)
	}

	err = file.CopyFile(src, dst)
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	data, err := ioutil.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read file %v", err)
	}
	gotContent := string(data)
	if gotContent != content {
		t.Errorf("got %q, want %q", gotContent, content)
	}
}

func TestCopyFileNotRegularFile(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "src")
	dst := filepath.Join(tmpdir, "dst")
	err := os.MkdirAll(src, 0o755)
	if err != nil {
		t.Fatalf("failed to create dir %v", err)
	}

	err = file.CopyFile(src, dst)
	if !errors.Is(err, file.ErrNotRegularFile) {
		t.Errorf("got %v err, want %v", err, file.ErrNotRegularFile)
	}
}

func TestCopyDirContents(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "src")
	dst := filepath.Join(tmpdir, "dst")
	err := os.MkdirAll(filepath.Join(src, "foodir"), 0o755)
	if err != nil {
		t.Fatalf("failed to create dir: %s", err)
	}
	const barfileContent = "bar"
	err = ioutil.WriteFile(filepath.Join(src, "barfile"), []byte(barfileContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}
	const bazfileContent = "baz"
	err = ioutil.WriteFile(filepath.Join(src, "foodir", "bazfile"), []byte(bazfileContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	err = file.CopyDirContents(src, dst)
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}

	data, err := ioutil.ReadFile(filepath.Join(dst, "barfile"))
	if err != nil {
		t.Fatalf("failed to read file %v", err)
	}
	gotContent := string(data)
	if gotContent != barfileContent {
		t.Errorf("got %q, want %q", gotContent, barfileContent)
	}

	data, err = ioutil.ReadFile(filepath.Join(dst, "foodir", "bazfile"))
	if err != nil {
		t.Fatalf("failed to read file %v", err)
	}
	gotContent = string(data)
	if gotContent != bazfileContent {
		t.Errorf("got %q, want %q", gotContent, bazfileContent)
	}
}

func TestCopyDirContentsNotDir(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "src")
	dst := filepath.Join(tmpdir, "dst")
	const content = `this is some file content`
	err := ioutil.WriteFile(src, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to write file %v", err)
	}

	err = file.CopyDirContents(src, dst)
	if !errors.Is(err, file.ErrNotDir) {
		t.Errorf("got %v err, want %v", err, file.ErrNotDir)
	}
}

func TestDirSize(t *testing.T) {
	tmpdir := t.TempDir()
	err := os.Mkdir(filepath.Join(tmpdir, "foodir"), 0o755)
	if err != nil {
		t.Fatalf("failed to create dir: %s", err)
	}
	const barfileContent = "bar"
	err = ioutil.WriteFile(filepath.Join(tmpdir, "barfile"), []byte(barfileContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}
	const bazfileContent = "baz"
	err = ioutil.WriteFile(filepath.Join(tmpdir, "foodir", "bazfile"), []byte(bazfileContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	size, err := file.DirSize(tmpdir)
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	// Size should be at least this much, but allow it to be more
	// in case there are filesystem shenanigans
	minSize := int64(len(barfileContent) + len(bazfileContent))
	if size < minSize {
		t.Errorf("got dir size %d, want it to be min %d", size, minSize)
	}
}

func TestDirSizeNotDir(t *testing.T) {
	tmpdir := t.TempDir()
	const barfileContent = "bar"
	barfilePath := filepath.Join(tmpdir, "barfile")
	err := ioutil.WriteFile(barfilePath, []byte(barfileContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	size, err := file.DirSize(barfilePath)
	if !errors.Is(err, file.ErrNotDir) {
		t.Errorf("got %v err, want %v", err, file.ErrNotDir)
	}
	if size != 0 {
		t.Errorf("got %d, want 0", size)
	}
}

func TestDirLen(t *testing.T) {
	tmpdir := t.TempDir()
	err := os.Mkdir(filepath.Join(tmpdir, "foodir"), 0o755)
	if err != nil {
		t.Fatalf("failed to create dir: %s", err)
	}
	err = ioutil.WriteFile(filepath.Join(tmpdir, "barfile"), []byte("bar"), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}
	err = ioutil.WriteFile(filepath.Join(tmpdir, "bazfile"), []byte("baz"), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	n, err := file.DirLen(tmpdir)
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	want := 3
	if n != want {
		t.Errorf("got dir len %d, want %d", n, want)
	}
}
