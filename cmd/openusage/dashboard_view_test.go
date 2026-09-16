package main

import (
	"testing"

	"github.com/janekbaraniewski/openusage/internal/config"
)

func TestResolveDashboardViewFlag(t *testing.T) {
	known := []string{
		config.DashboardViewGrid,
		config.DashboardViewStacked,
		config.DashboardViewTabs,
		config.DashboardViewSplit,
		config.DashboardViewCompare,
		config.DashboardViewCompact,
	}
	for _, name := range known {
		if got, err := resolveDashboardViewFlag(name, known); err != nil || got != name {
			t.Errorf("resolveDashboardViewFlag(%q) = (%q, %v)", name, got, err)
		}
	}
	// Case-insensitive, whitespace-tolerant.
	if got, err := resolveDashboardViewFlag("  Compact ", known); err != nil || got != config.DashboardViewCompact {
		t.Errorf("resolveDashboardViewFlag(%q) = (%q, %v)", "  Compact ", got, err)
	}
	// Unknown values must error, never silently fall back.
	if _, err := resolveDashboardViewFlag("tiles", known); err == nil {
		t.Error("resolveDashboardViewFlag(tiles) expected an error")
	}
}
