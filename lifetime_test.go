package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLifetimeZeroValueIsNotPermanent(t *testing.T) {
	var forgotten lifetime
	require.False(t, forgotten.isPermanent(), "a lifetime nobody set must not become a permanent mapping")
	require.False(t, forgotten.valid())
}

func TestLifetimeValidity(t *testing.T) {
	require.True(t, permanentLifetime.valid())
	require.True(t, permanentLifetime.isPermanent())
	require.True(t, remainingLifetime(time.Nanosecond).valid())
	require.False(t, remainingLifetime(time.Nanosecond).isPermanent())
	require.False(t, remainingLifetime(0).valid())
	require.False(t, remainingLifetime(-time.Second).valid())
}

func TestLifetimeCappedAt(t *testing.T) {
	for _, tc := range []struct {
		name string
		life lifetime
		max  time.Duration
		want time.Duration
	}{
		{"limited by the cap", remainingLifetime(365 * 24 * time.Hour), 5 * time.Minute, 5 * time.Minute},
		{"limited by the lifetime", remainingLifetime(time.Minute), 5 * time.Minute, time.Minute},
		{"permanent takes the cap", permanentLifetime, 5 * time.Minute, 5 * time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.life.cappedAt(tc.max))
		})
	}
}
