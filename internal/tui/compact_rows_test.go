package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janekbaraniewski/openusage/internal/core"
)

func fptr(v float64) *float64 { return &v }

func compactTestSnapshots() map[string]core.UsageSnapshot {
	now := time.Now()
	return map[string]core.UsageSnapshot{
		"muse": {
			ProviderID: "muse_code",
			AccountID:  "muse",
			Status:     core.StatusOK,
			Metrics: map[string]core.Metric{
				"usage_five_hour": {Used: fptr(42), Unit: "%"},
				"usage_seven_day": {Used: fptr(49), Unit: "%"},
			},
			Resets: map[string]time.Time{
				"usage_five_hour": now.Add(90 * time.Minute),
				"usage_seven_day": now.Add(72 * time.Hour),
			},
		},
	}
}

func TestNormalizeDashboardViewMode_Compact(t *testing.T) {
	if got := normalizeDashboardViewMode("compact"); got != dashboardViewCompact {
		t.Fatalf("normalize = %q, want compact", got)
	}
	if got := dashboardViewLabel(dashboardViewCompact); got != "Compact" {
		t.Fatalf("label = %q, want Compact", got)
	}
}

func TestHandleKey_CyclesIntoAndOutOfCompact(t *testing.T) {
	m := Model{dashboardView: dashboardViewCompare, screen: screenDashboard}
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	if got := updated.(Model).dashboardView; got != dashboardViewCompact {
		t.Fatalf("after compare = %q, want compact", got)
	}
	updated, _ = updated.(Model).handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	if got := updated.(Model).dashboardView; got != dashboardViewGrid {
		t.Fatalf("after compact = %q, want wrap to grid", got)
	}
}

func TestRenderCompactRows_TwoNumbersPerProvider(t *testing.T) {
	m := Model{
		dashboardView: dashboardViewCompact,
		width:         80,
		sortedIDs:     []string{"muse"},
		snapshots:     compactTestSnapshots(),
	}
	out := m.renderCompactRows(80, 24)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("rows = %d, want one row per provider:\n%s", len(lines), out)
	}
	for _, want := range []string{"muse", "42%", "49%", "1h30m", "3d00h"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("row missing %q:\n%s", want, lines[0])
		}
	}
}

func TestRenderCompactRows_NoProviders(t *testing.T) {
	m := Model{dashboardView: dashboardViewCompact, width: 80}
	if out := m.renderCompactRows(80, 24); !strings.Contains(out, "No providers") {
		t.Errorf("empty state = %q", out)
	}
}

func TestNormalizeCompactGlyphs(t *testing.T) {
	cases := map[string]string{
		"": "off", "off": "off", "bogus": "off",
		"unicode":  "unicode",
		"nerdfont": "nerdfont", "nerd": "nerdfont", "nf": "nerdfont",
		"customfont": "customfont", "custom": "customfont",
		"ascii": "ascii",
	}
	for in, want := range cases {
		if got := normalizeCompactGlyphs(in); got != want {
			t.Errorf("normalizeCompactGlyphs(%q) = %q, want %q", in, got, want)
		}
	}
}

func compactGlyphModel(glyphs string) Model {
	m := Model{
		dashboardView: dashboardViewCompact,
		width:         80,
		sortedIDs:     []string{"muse", "copilot"},
		snapshots:     compactTestSnapshots(),
		compactGlyphs: normalizeCompactGlyphs(glyphs),
	}
	copilot := m.snapshots["muse"]
	copilot.ProviderID = "copilot"
	copilot.AccountID = "copilot"
	m.snapshots["copilot"] = copilot
	return m
}

func TestRenderCompactRows_IconsOffByDefault(t *testing.T) {
	out := compactGlyphModel("").renderCompactRows(80, 24)
	for _, banned := range []string{"🤖", "🧠", "[copilot]"} {
		if strings.Contains(out, banned) {
			t.Errorf("default row must not carry logos, found %q:\n%s", banned, out)
		}
	}
}

func TestRenderCompactRows_UnicodeLogos(t *testing.T) {
	out := compactGlyphModel("unicode").renderCompactRows(80, 24)
	// muse_code -> robot, copilot -> brain; segments unchanged.
	for _, want := range []string{"🤖", "🧠", "muse", "copilot", "42%", "49%"} {
		if !strings.Contains(out, want) {
			t.Errorf("unicode row missing %q:\n%s", want, out)
		}
	}
}

func TestRenderCompactRows_AsciiLabelsReplaceName(t *testing.T) {
	out := compactGlyphModel("ascii").renderCompactRows(80, 24)
	if !strings.Contains(out, "[copilot]") {
		t.Fatalf("ascii row missing bracket label:\n%s", out)
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.Contains(line, "[copilot]") && strings.Contains(line, "copilot  ") {
			t.Errorf("ascii label must replace the name, not repeat it:\n%s", line)
		}
	}
}

func TestRenderCompactRows_CustomFontGlyph(t *testing.T) {
	// muse_code aliases to the bundled claude glyph (U+E900) in the
	// customfont tier — no generic-sparkles degradation for the flagship.
	out := compactGlyphModel("customfont").renderCompactRows(80, 24)
	if !strings.Contains(out, "\ue900") {
		t.Errorf("customfont row for muse should carry the bundled Claude glyph:\n%s", out)
	}
}

func TestRenderCompactRows_CustomFontFallback(t *testing.T) {
	// Providers with no bundled glyph at all (azure_openai) degrade to the
	// unicode emoji rather than emitting a blank or tofu box.
	snap := compactTestSnapshots()["muse"]
	snap.ProviderID = "azure_openai"
	snap.AccountID = "azure"
	m := Model{
		dashboardView: dashboardViewCompact,
		width:         80,
		sortedIDs:     []string{"azure"},
		snapshots:     map[string]core.UsageSnapshot{"azure": snap},
		compactGlyphs: "customfont",
	}
	out := m.renderCompactRows(80, 24)
	if !strings.Contains(out, "✨") {
		t.Errorf("customfont row for glyph-less provider should fall back to emoji:\n%s", out)
	}
}

func TestRenderCompactRows_UnknownPercent(t *testing.T) {
	snap := compactTestSnapshots()["muse"]
	snap.Metrics = nil // resets without backing metrics
	m := Model{
		dashboardView: dashboardViewCompact,
		width:         80,
		sortedIDs:     []string{"muse"},
		snapshots:     map[string]core.UsageSnapshot{"muse": snap},
	}
	out := m.renderCompactRows(80, 24)
	if !strings.Contains(out, "muse") || !strings.Contains(out, "no quota") {
		t.Errorf("unknown-percent row = %q", out)
	}
}
