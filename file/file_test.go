package file_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cszatmary/goutils/file"
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
	assertFile(t, downloadPath, content)
}

func TestCopyFile(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "src")
	dst := filepath.Join(tmpdir, "dst")
	const content = `this is some file content`
	err := os.WriteFile(src, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to write file %v", err)
	}

	err = file.CopyFile(src, dst)
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	assertFile(t, dst, content)
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
	err = os.WriteFile(filepath.Join(src, "barfile"), []byte(barfileContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}
	const bazfileContent = "baz"
	err = os.WriteFile(filepath.Join(src, "foodir", "bazfile"), []byte(bazfileContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	err = file.CopyDirContents(src, dst)
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	assertFile(t, filepath.Join(dst, "barfile"), barfileContent)
	assertFile(t, filepath.Join(dst, "foodir", "bazfile"), bazfileContent)
}

func TestCopyDirContentsNotDir(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "src")
	dst := filepath.Join(tmpdir, "dst")
	const content = `this is some file content`
	err := os.WriteFile(src, []byte(content), 0o644)
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
	err = os.WriteFile(filepath.Join(tmpdir, "barfile"), []byte(barfileContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}
	const bazfileContent = "baz"
	err = os.WriteFile(filepath.Join(tmpdir, "foodir", "bazfile"), []byte(bazfileContent), 0o644)
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
	err := os.WriteFile(barfilePath, []byte(barfileContent), 0o644)
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
	err = os.WriteFile(filepath.Join(tmpdir, "barfile"), []byte("bar"), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}
	err = os.WriteFile(filepath.Join(tmpdir, "bazfile"), []byte("baz"), 0o644)
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

func TestUntar(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"normal tar file", "testdata/basic.tar"},
		{"gzip-compressed tar file", "testdata/basic.tgz"},
		{"tar file without directories", "testdata/basic_nodirs.tgz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Fatalf("failed to open %s: %v", tt.path, err)
			}
			t.Cleanup(func() {
				f.Close()
			})

			tmpdir := t.TempDir()
			err = file.Untar(tmpdir, f)
			if err != nil {
				t.Fatalf("want nil error, got %v", err)
			}

			assertFile(t, filepath.Join(tmpdir, "a.txt"), "This is a file\n")
			// This means the b dir exists by definition
			assertFile(t, filepath.Join(tmpdir, "b/c.txt"), "This is another file inside a directory\n")
		})
	}
}

func TestUntarSymlink(t *testing.T) {
	const path = "testdata/basic_symlink.tgz"
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open %s: %v", path, err)
	}
	t.Cleanup(func() {
		f.Close()
	})

	tmpdir := t.TempDir()
	err = file.Untar(tmpdir, f)
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}

	assertFile(t, filepath.Join(tmpdir, "a.txt"), "This is a file\n")
	// Check that symlink was created with the right path
	cPath := filepath.Join(tmpdir, "b/c.txt")
	link, err := os.Readlink(cPath)
	if err != nil {
		t.Fatalf("failed to read link %s: %v", cPath, err)
	}
	const wantLink = "../a.txt"
	if link != wantLink {
		t.Errorf("got symlink %q, want %q", link, wantLink)
	}
	assertFile(t, cPath, "This is a file\n")
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	got := string(b)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
