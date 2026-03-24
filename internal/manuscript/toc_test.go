package manuscript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOC_BasicChapters(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	bookYAML := `title: Test Novel
author: Test Author
---
book:
  base_dir: scenes
  chapters:
    - scenes:
        - opening
        - cafe-meeting
    - scenes:
        - confrontation
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))

	var buf strings.Builder
	err := TOC(&buf)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 5)
	assert.Equal(t, "1\tChapter 1", lines[0])
	assert.Equal(t, "1.1\topening", lines[1])
	assert.Equal(t, "1.2\tcafe-meeting", lines[2])
	assert.Equal(t, "2\tChapter 2", lines[3])
	assert.Equal(t, "2.1\tconfrontation", lines[4])
}

func TestTOC_NamedChapters(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	bookYAML := `title: Test Novel
author: Test Author
---
book:
  base_dir: scenes
  chapters:
    - name: The Beginning
      scenes:
        - opening
    - name: The End
      scenes:
        - finale
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))

	var buf strings.Builder
	err := TOC(&buf)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 4)
	assert.Equal(t, "1\tThe Beginning", lines[0])
	assert.Equal(t, "1.1\topening", lines[1])
	assert.Equal(t, "2\tThe End", lines[2])
	assert.Equal(t, "2.1\tfinale", lines[3])
}

func TestTOC_Interludes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	bookYAML := `title: Test Novel
author: Test Author
---
book:
  base_dir: scenes
  chapters:
    - scenes:
        - opening
    - interlude: true
      scenes:
        - flashback
    - scenes:
        - resolution
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))

	var buf strings.Builder
	err := TOC(&buf)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 6)
	assert.Equal(t, "1\tChapter 1", lines[0])
	assert.Equal(t, "1.1\topening", lines[1])
	assert.Equal(t, "2\tInterlude", lines[2])
	assert.Equal(t, "2.1\tflashback", lines[3])
	assert.Equal(t, "3\tChapter 2", lines[4])
	assert.Equal(t, "3.1\tresolution", lines[5])
}

func TestTOC_EmptyBook(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	bookYAML := `title: Test Novel
author: Test Author
---
book:
  base_dir: scenes
  chapters: []
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0644))

	var buf strings.Builder
	err := TOC(&buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestChapterHeading(t *testing.T) {
	assert.Equal(t, "My Chapter", chapterHeading("My Chapter", false, 1))
	assert.Equal(t, "Named Interlude", chapterHeading("Named Interlude", true, 2))
	assert.Equal(t, "Interlude", chapterHeading("", true, 3))
	assert.Equal(t, "Chapter 5", chapterHeading("", false, 5))
}
