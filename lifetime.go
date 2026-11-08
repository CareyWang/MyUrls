package main

import (
	"errors"
	"time"
)

var errInvalidLifetime = errors.New("lifetime must be permanent or positive")

// lifetime is how long a mapping stays valid: permanent, or a positive duration.
// The zero value is neither, so a lifetime nobody set cannot become a permanent
// mapping. A mapping that is already gone has no lifetime, and stores report it
// as errShortURLNotFound instead.
type lifetime struct {
	permanent bool
	remaining time.Duration
}

var permanentLifetime = lifetime{permanent: true}

// remainingLifetime is invalid unless d is positive.
func remainingLifetime(d time.Duration) lifetime {
	return lifetime{remaining: d}
}

func (l lifetime) isPermanent() bool {
	return l.permanent
}

func (l lifetime) valid() bool {
	return l.permanent || l.remaining > 0
}

// cappedAt returns how long to keep something that must not outlive the mapping
// or max, whichever ends first.
func (l lifetime) cappedAt(max time.Duration) time.Duration {
	if l.permanent {
		return max
	}
	return min(l.remaining, max)
}
