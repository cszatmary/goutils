// Package file provides various helpers for working with files on an OS filesystem.
package file

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const mkdirDefaultPerms = 0o755

// ErrNotDir indicates a path was not a directory.
var ErrNotDir = errors.New("not a directory")

// ErrNotRegularFile indicates a path was not a regular file.
// For example, this could mean it was a symlink.
var ErrNotRegularFile = errors.New("not a regular file")

// Exists checks if a file or directory exists at path.
func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// Download creates or replaces a file at downloadPath by reading from r.
func Download(downloadPath string, r io.Reader) (int64, error) {
	// Check if file exists
	downloadDir := filepath.Dir(downloadPath)
	if err := os.MkdirAll(downloadDir, mkdirDefaultPerms); err != nil {
		return 0, fmt.Errorf("failed to create directory %q: %w", downloadDir, err)
	}

	// Write payload to target dir
	f, err := os.Create(downloadPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create file %q: %w", downloadPath, err)
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return 0, fmt.Errorf("failed writing data to file %q: %w", downloadPath, err)
	}
	return n, nil
}

// CopyFile copies the regular file located at src to dst. Any intermediate directories in dst
// that do not exists will be created. If src is not a regular file an error will be returned.
func CopyFile(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("failed to get info of %q: %w", src, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %q", ErrNotRegularFile, src)
	}
	return copyFile(src, dst, info)
}

// copyFile is the actual implementation of CopyFile. It assumes that src
// has already been verified to be a regular file.
func copyFile(src, dst string, info os.FileInfo) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, mkdirDefaultPerms); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("failed to open/create file %q: %w", dst, err)
	}
	defer f.Close()

	s, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %q: %w", src, err)
	}
	defer s.Close()

	if _, err = io.Copy(f, s); err != nil {
		return fmt.Errorf("failed to copy %q to %q: %w", src, dst, err)
	}
	return nil
}

// CopyDirContents copies all contents from the directory src to the directory dst.
// Only regular files and directories will be copied. If src or dst is not a directory,
// and error will be returned. If dst does not exists, it will be created.
func CopyDirContents(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("failed to get info of %q: %w", src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %q", ErrNotDir, src)
	}
	return copyDirContents(src, dst, info)
}

// copyDirContents is the actual implementation of CopyDirContents. It assumes that src
// has already been verified to be a directory file.
func copyDirContents(src, dst string, info os.FileInfo) error {
	// Make sure dst exists, if it does this is a no-op
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dst, err)
	}

	contents, err := ioutil.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read contents of directory %q: %w", src, err)
	}

	for _, item := range contents {
		srcItemPath := filepath.Join(src, item.Name())
		dstItemPath := filepath.Join(dst, item.Name())

		if item.IsDir() {
			err := copyDirContents(srcItemPath, dstItemPath, item)
			if err != nil {
				return fmt.Errorf("failed to copy directory %q: %w", srcItemPath, err)
			}
			continue
		}
		if !item.Mode().IsRegular() {
			// Unsupported file type, ignore
			continue
		}

		err := copyFile(srcItemPath, dstItemPath, item)
		if err != nil {
			return fmt.Errorf("failed to copy file %q: %w", srcItemPath, err)
		}
	}
	return nil
}

// DirSize returns the size of the directory located at path.
func DirSize(path string) (int64, error) {
	s, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if !s.IsDir() {
		return 0, fmt.Errorf("%w: %q", ErrNotDir, path)
	}

	var size int64
	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// DirLen returns the number of items in the directory located at path.
func DirLen(path string) (int, error) {
	dir, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	list, err := dir.Readdirnames(0)
	return len(list), err
}
