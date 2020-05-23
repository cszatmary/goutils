package file

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirExists(t *testing.T) {
	path := "testdata/text_tests"
	assert.True(t, FileOrDirExists(path))
}

func TestFileExists(t *testing.T) {
	path := "testdata/text_tests/hype.md"
	assert.True(t, FileOrDirExists(path))
}

func TestFileNotExists(t *testing.T) {
	path := "testdata/notafile.txt"
	assert.False(t, FileOrDirExists(path))
}
