package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString_Defaults(t *testing.T) {
	assert.Equal(t, "scrib dev (unknown) unknown", String())
}

func TestString_WithValues(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	})

	Version = "v1.2.3"
	Commit = "abc1234"
	Date = "2026-03-20"

	assert.Equal(t, "scrib v1.2.3 (abc1234) 2026-03-20", String())
}
