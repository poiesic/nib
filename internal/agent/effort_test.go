package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEffort(t *testing.T) {
	cases := []struct {
		in      string
		want    Effort
		wantErr bool
	}{
		{"", DefaultEffort, false},
		{"low", EffortLow, false},
		{"medium", EffortMedium, false},
		{"high", EffortHigh, false},
		{"xhigh", EffortXHigh, false},
		{"max", EffortMax, false},
		{"LOW", "", true},
		{"extreme", "", true},
		{" high ", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ValidateEffort(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEffortBelowHigh(t *testing.T) {
	assert.True(t, EffortLow.BelowHigh())
	assert.True(t, EffortMedium.BelowHigh())
	assert.False(t, EffortHigh.BelowHigh())
	assert.False(t, EffortXHigh.BelowHigh())
	assert.False(t, EffortMax.BelowHigh())
}

func TestDefaultEffortIsAtLeastHigh(t *testing.T) {
	assert.False(t, DefaultEffort.BelowHigh(), "default effort should not trigger the below-high warning")
}
