package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/poiesic/nib/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEffortValue_EmptyDefaults(t *testing.T) {
	var buf bytes.Buffer
	got, err := resolveEffortValue("", &buf)
	require.NoError(t, err)
	assert.Equal(t, agent.DefaultEffort, got)
	assert.Empty(t, buf.String(), "empty input should not emit a warning")
}

func TestResolveEffortValue_WarnsBelowHigh(t *testing.T) {
	for _, level := range []string{"low", "medium"} {
		t.Run(level, func(t *testing.T) {
			var buf bytes.Buffer
			got, err := resolveEffortValue(level, &buf)
			require.NoError(t, err)
			assert.Equal(t, agent.Effort(level), got)
			assert.Contains(t, buf.String(), "warning")
			assert.Contains(t, buf.String(), level)
		})
	}
}

func TestResolveEffortValue_NoWarnAtOrAboveHigh(t *testing.T) {
	for _, level := range []string{"high", "xhigh", "max"} {
		t.Run(level, func(t *testing.T) {
			var buf bytes.Buffer
			got, err := resolveEffortValue(level, &buf)
			require.NoError(t, err)
			assert.Equal(t, agent.Effort(level), got)
			assert.Empty(t, buf.String())
		})
	}
}

func TestResolveEffortValue_InvalidErrors(t *testing.T) {
	var buf bytes.Buffer
	_, err := resolveEffortValue("extreme", &buf)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid effort"))
}
