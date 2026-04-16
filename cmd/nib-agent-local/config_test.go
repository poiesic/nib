package main

import (
	"testing"

	"github.com/poiesic/nib/internal/agent"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_EffortTemperature(t *testing.T) {
	cases := []struct {
		name   string
		effort agent.Effort
		want   float64
	}{
		{"low", agent.EffortLow, 0.3},
		{"medium", agent.EffortMedium, 0.5},
		{"high", agent.EffortHigh, 0.7},
		{"xhigh", agent.EffortXHigh, 0.8},
		{"max", agent.EffortMax, 0.9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadConfig(tc.effort)
			assert.InDelta(t, tc.want, cfg.Temperature, 0.0001)
		})
	}
}

func TestLoadConfig_EmptyEffortUsesDefault(t *testing.T) {
	cfg := loadConfig("")
	assert.InDelta(t, effortTemperature[agent.DefaultEffort], cfg.Temperature, 0.0001)
}

func TestLoadConfig_UnknownEffortUsesDefault(t *testing.T) {
	cfg := loadConfig(agent.Effort("bogus"))
	assert.InDelta(t, effortTemperature[agent.DefaultEffort], cfg.Temperature, 0.0001)
}

func TestLoadConfig_MaxTemperatureIsCappedAtPointNine(t *testing.T) {
	for _, e := range agent.AllEfforts() {
		cfg := loadConfig(e)
		assert.LessOrEqual(t, cfg.Temperature, 0.9, "temperature for %s exceeds 0.9", e)
	}
}
