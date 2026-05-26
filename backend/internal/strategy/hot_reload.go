// Package strategy provides hot-reload support for strategy instances (M10-BASE-E0).
//
// Hot-reload flow:
//
//	old instance → OnStop() → Snapshot() → PG strategy_state
//	new instance → load from PG → Restore() → OnStart(state)
//
// Version compatibility: major version mismatch → reject hot-reload → force rollover
// (old runs to bar close, new starts fresh at next bar).
package strategy

import (
	"fmt"
	"strconv"
	"strings"
)

// VersionCompatibility represents the result of comparing two strategy versions.
type VersionCompatibility int

const (
	VersionCompatible       VersionCompatibility = iota // same version or compatible patch bump
	VersionMinorCompatible                              // minor bump, state may need migration
	VersionMajorIncompatible                            // major bump, hot-reload rejected
)

// CheckVersionCompatibility compares two semver versions.
// Returns MajorIncompatible if the major versions differ.
func CheckVersionCompatibility(oldVersion, newVersion string) VersionCompatibility {
	old := parseSemver(oldVersion)
	new := parseSemver(newVersion)

	if old.major != new.major {
		return VersionMajorIncompatible
	}
	if old.minor != new.minor {
		return VersionMinorCompatible
	}
	return VersionCompatible
}

type semver struct {
	major, minor, patch int
}

func parseSemver(v string) semver {
	// Strip leading 'v' if present.
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	s := semver{}
	if len(parts) > 0 {
		s.major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		s.minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		s.patch, _ = strconv.Atoi(parts[2])
	}
	return s
}

// HotReloadError is returned when hot-reload fails (version mismatch, state corruption, etc).
type HotReloadError struct {
	Reason      string
	OldVersion  string
	NewVersion  string
	CanRollover bool // true if old instance can continue while new starts fresh
}

func (e *HotReloadError) Error() string {
	return fmt.Sprintf("strategy: hot-reload failed: %s (old=%s new=%s rollover=%v)",
		e.Reason, e.OldVersion, e.NewVersion, e.CanRollover)
}

// ErrVersionMismatch creates a hot-reload error for incompatible versions.
func ErrVersionMismatch(oldVersion, newVersion string) error {
	return &HotReloadError{
		Reason:      "major version mismatch, incompatible state schema",
		OldVersion:  oldVersion,
		NewVersion:  newVersion,
		CanRollover: true,
	}
}

// Reloader manages strategy hot-reload with version checking.
type Reloader struct {
	current Strategy
}

// NewReloader creates a hot-reload manager for a strategy.
func NewReloader(s Strategy) *Reloader {
	return &Reloader{current: s}
}

// Current returns the currently active strategy.
func (r *Reloader) Current() Strategy { return r.current }

// ValidateUpgrade checks if a new strategy version is compatible with the current one.
func (r *Reloader) ValidateUpgrade(newStrategy Strategy) error {
	compat := CheckVersionCompatibility(r.current.Version(), newStrategy.Version())
	if compat == VersionMajorIncompatible {
		return ErrVersionMismatch(r.current.Version(), newStrategy.Version())
	}
	return nil
}

// Reload attempts to hot-reload from current to new strategy.
// On major version mismatch, returns HotReloadError and keeps current running.
func (r *Reloader) Reload(newStrategy Strategy, state *StrategyState) error {
	if err := r.ValidateUpgrade(newStrategy); err != nil {
		return err
	}
	// Minor compatible: state may be migrated (future enhancement).
	if err := newStrategy.Restore(state.CustomState); err != nil {
		return &HotReloadError{
			Reason:      fmt.Sprintf("state restore failed: %v", err),
			OldVersion:  r.current.Version(),
			NewVersion:  newStrategy.Version(),
			CanRollover: true,
		}
	}
	r.current = newStrategy
	return nil
}
