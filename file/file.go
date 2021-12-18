// Package file provides various helpers for working with files on an OS filesystem.
package file

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
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

// Download creates or replaces a file at dst by reading from r.
func Download(dst string, r io.Reader) (int64, error) {
	// Check if file exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, mkdirDefaultPerms); err != nil {
		return 0, fmt.Errorf("failed to create directory %q: %w", dstDir, err)
	}

	// Write payload to target dir
	f, err := os.Create(dst)
	if err != nil {
		return 0, fmt.Errorf("failed to create file %q: %w", dst, err)
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return 0, fmt.Errorf("failed writing data to file %q: %w", dst, err)
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

	contents, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read contents of directory %q: %w", src, err)
	}

	for _, item := range contents {
		srcItemPath := filepath.Join(src, item.Name())
		dstItemPath := filepath.Join(dst, item.Name())
		fi, err := item.Info()
		if err != nil {
			return fmt.Errorf("failed to get info of %q: %w", srcItemPath, err)
		}

		if item.IsDir() {
			err := copyDirContents(srcItemPath, dstItemPath, fi)
			if err != nil {
				return fmt.Errorf("failed to copy directory %q: %w", srcItemPath, err)
			}
			continue
		}
		if !fi.Mode().IsRegular() {
			// Unsupported file type, ignore
			continue
		}
		if err := copyFile(srcItemPath, dstItemPath, fi); err != nil {
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

// Untar reads the tar file from r and writes it to dir.
// It can handle gzip-compressed tar files.
//
// Note that Untar will overwrite any existing files with the same path
// as files in the archive.
func Untar(dir string, r io.Reader) error {
	// Determine if we are dealing with a gzip-compressed tar file.
	// gzip files are identified by the first 3 bytes.
	// See section 2.3.1. of RFC 1952: https://www.ietf.org/rfc/rfc1952.txt
	buf := make([]byte, 3)
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("unable to check if tar file is gzip-compressed: %w", err)
	}

	// Need to create a new reader with the 3 bytes added back to move back to the
	// start of the file. Can do this by concatenating buf with r.
	rr := io.MultiReader(bytes.NewReader(buf), r)
	if buf[0] == 0x1f && buf[1] == 0x8b && buf[2] == 8 {
		gzr, err := gzip.NewReader(rr)
		if err != nil {
			return fmt.Errorf("unable to read gzip-compressed tar file: %w", err)
		}
		defer gzr.Close()
		rr = gzr
	}
	tr := tar.NewReader(rr)

	// Now we get to the fun part, the actual tar extraction.
	// Loop through each entry in the archive and extract it.
	// Keep track of a list of dirs created so we don't waste time creating the same dir multiple times.
	madeDirs := make(map[string]struct{})
	for {
		header, err := tr.Next()
		if err == io.EOF {
			// End of the archive, we are done.
			return nil
		} else if err != nil {
			return fmt.Errorf("untar: read error: %w", err)
		}

		dst := filepath.Join(dir, header.Name)
		// Ensure the parent directory exists. Usually this shouldn't be required since there
		// should be a directory entry in the tar file that created the directory beforehand.
		// However, testing has revealed that this is not always the case and there can be
		// tar files without directory entries so we should handle those cases.
		parentDir := filepath.Dir(dst)
		if _, ok := madeDirs[parentDir]; !ok {
			if err := os.MkdirAll(parentDir, mkdirDefaultPerms); err != nil {
				return fmt.Errorf("untar: create directory error: %w", err)
			}
			madeDirs[parentDir] = struct{}{}
		}

		mode := header.FileInfo().Mode()
		switch {
		case mode.IsDir():
			if err := os.MkdirAll(dst, mkdirDefaultPerms); err != nil {
				return fmt.Errorf("untar: create directory error: %w", err)
			}
			// Mark the dir as created so files in this dir don't need to create it again.
			madeDirs[dst] = struct{}{}
		case mode.IsRegular():
			// Now we can create the actual file. Untar will overwrite any existing files.
			f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return fmt.Errorf("untar: create file error: %w", err)
			}
			n, err := io.Copy(f, tr)

			// We need to manually close the file here instead of using defer since defer runs when
			// the function exits and would cause all files to remain open until this loop is finished.
			if closeErr := f.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				return fmt.Errorf("untar: error writing file to %s: %w", dst, err)
			}
			// Make sure the right amount of bytes were written just to be safe.
			if n != header.Size {
				return fmt.Errorf("untar: only wrote %d bytes to %s; expected %d", n, dst, header.Size)
			}
		case mode&os.ModeSymlink != 0:
			// Entry is a symlink, need to create a symlink to the target
			if err := os.Symlink(header.Linkname, dst); err != nil {
				return fmt.Errorf("untar: symlink error: %w", err)
			}
		default:
			return fmt.Errorf("tar file entry %s has unsupported file type %v", header.Name, mode)
		}
	}
}
