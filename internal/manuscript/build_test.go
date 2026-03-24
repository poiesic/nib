package manuscript

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCmd_Docx(t *testing.T) {
	chapterFiles := []string{"build/001.md"}
	cmd, err := buildCmd(FormatDocx, mockRunner, "/project", "build", "novel", chapterFiles)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
}

func TestBuildCmd_PDF(t *testing.T) {
	chapterFiles := []string{"build/001.md"}
	cmd, err := buildCmd(FormatPDF, mockRunner, "/project", "build", "novel", chapterFiles)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
}

func TestBuildCmd_EPUB(t *testing.T) {
	chapterFiles := []string{"build/001.md"}
	cmd, err := buildCmd(FormatEPUB, mockRunner, "/project", "build", "novel", chapterFiles)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
}

func TestBuildCmd_Unknown(t *testing.T) {
	chapterFiles := []string{"build/001.md"}
	cmd, err := buildCmd(Format("unknown"), mockRunner, "/project", "build", "novel", chapterFiles)
	assert.NoError(t, err)
	assert.Nil(t, cmd)
}
