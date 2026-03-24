package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

type config struct {
	Endpoint      string
	Model         string
	Temperature   float64
	RepeatPenalty float64
	TopP          float64
	MaxTokens     int
	NoThink       bool
}

const (
	defaultEndpoint = "http://localhost:1234/v1"
	defaultModel    = "qwen3-30b-a3b"
)

// Effort-based temperature presets
var effortTemperature = map[string]float64{
	"low":    0.3,
	"medium": 0.5,
	"high":   0.7,
}

func loadConfig(effort string) config {
	cfg := config{
		Endpoint:      resolve("NIB_LOCAL_ENDPOINT", "nib.agents.local.endpoint", defaultEndpoint),
		Model:         resolve("NIB_LOCAL_MODEL", "nib.agents.local.model", defaultModel),
		RepeatPenalty: 1.1,
		TopP:          0.9,
		MaxTokens:     4096,
		NoThink:       resolve("NIB_LOCAL_NO_THINK", "nib.agents.local.no-think", "true") == "true",
	}

	// Set temperature from effort level
	if t, ok := effortTemperature[effort]; ok {
		cfg.Temperature = t
	} else {
		cfg.Temperature = 0.5
	}

	return cfg
}

// resolve checks env var, then git config, then returns the default.
func resolve(envKey, gitKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	cmd := exec.Command("git", "config", "--get", gitKey)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		if v := strings.TrimSpace(out.String()); v != "" {
			return v
		}
	}
	return fallback
}
