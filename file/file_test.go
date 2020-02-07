package file

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const fixtures = "../_fixtures"
const textTests = fixtures + "/text_tests"

func TestDirExists(t *testing.T) {
	path := textTests
	assert.True(t, FileOrDirExists(path))
}

func TestFileExists(t *testing.T) {
	path := textTests + "/hype.md"
	assert.True(t, FileOrDirExists(path))
}

func TestFileNotExists(t *testing.T) {
	path := fixtures + "/notafile.txt"
	assert.False(t, FileOrDirExists(path))
}
