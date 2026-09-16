package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/janekbaraniewski/openusage/internal/core"
	"github.com/janekbaraniewski/openusage/internal/tmux"
)

// normalizeCompactGlyphs canonicalizes the dashboard.compact_icons setting.
// Unknown and empty values mean "off": today's status-shape row, no font
// needed anywhere.
func normalizeCompactGlyphs(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "unicode":
		return "unicode"
	case "nerdfont", "nerd", "nf":
		return "nerdfont"
	case "customfont", "custom", "openusage":
		return "customfont"
	case "ascii":
		return "ascii"
	default:
		return "off"
	}
}

// compactLogoPrefix resolves the provider logo for a glyph tier and folds it
// into the row head. The ascii tier glyph is a bracketed label, so it takes
// the name's place instead of prepending. Unknown providers degrade through
// ProviderIcon's own fallback chain and never come back blank.
func compactLogoPrefix(providerID, glyphs, head, nameStr string) (string, string) {
	tier := tmux.ParseGlyphTier(glyphs)
	logo := tmux.ProviderIcon(providerID, tier)
	if logo == "" {
		return head, nameStr
	}
	if brand := tmux.ProviderBrandColor(providerID); brand != "" && tier != tmux.GlyphTierUnicode {
		logo = lipgloss.NewStyle().Foreground(lipgloss.Color(brand)).Render(logo)
	}
	if tier == tmux.GlyphTierASCII {
		return head, logo
	}
	return head + logo + " ", nameStr
}

// Compact quota color thresholds (used percent): yellow at >=75%, red at >=90%.
const (
	compactWarnThresh = 0.25
	compactCritThresh = 0.10
)

// Compact dashboard view (#346): one dense row per provider — account name
// plus up to two quota segments (used percent + reset countdown), ordered by
// the same priority the tile reset pills use. No cards, no telemetry strip,
// deliberately label-free to fit narrow terminals: the segments follow reset
// priority (session-ish window first).

// renderCompactRows renders one row per visible provider.
func (m Model) renderCompactRows(w, contentH int) string {
	ids := m.filteredIDs()
	if len(ids) == 0 {
		return dimStyle.Render("No providers configured.")
	}
	_ = contentH // rows scroll; no height cap like tiles have.
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		snap, ok := m.snapshots[id]
		if !ok {
			continue
		}
		lines = append(lines, compactRow(snap, w, m.compactGlyphs))
	}
	if len(lines) == 0 {
		return dimStyle.Render("No providers configured.")
	}
	return strings.Join(lines, "\n")
}

// compactRow builds `name  42%·2h15m  49%·3d04h`. Percentages come from
// core.MetricUsedPercent so raw-unit providers (tokens/requests) and
// percent-unit providers (Muse quota) render on the same 0-100 scale;
// entries without a computable percent are skipped, and a provider with no
// quota data keeps its name with a dimmed placeholder.
//
// glyphs selects the optional provider logo (dashboard.compact_icons):
// "off" (default) keeps the status-shape row; otherwise a tier glyph from
// the shared tmux icon set is prepended — tinted with the provider brand
// color except in the unicode tier, where emoji keep native colors. The
// ascii tier glyph is itself a label ([copilot]), so it replaces the name.
func compactRow(snap core.UsageSnapshot, w int, glyphs string) string {
	widget := dashboardWidget(snap.ProviderID)
	name := snap.AccountID
	if name == "" {
		name = snap.ProviderID
	}
	// Same color vocabulary as tiles: status-colored icon, provider-colored
	// name, usage-colored percent (green/yellow/red), dimmed countdown.
	icon := StatusIcon(snap.Status)
	iconStr := lipgloss.NewStyle().Foreground(StatusColor(snap.Status)).Render(icon)
	nameStr := lipgloss.NewStyle().Bold(true).Foreground(ProviderColor(snap.ProviderID)).Render(name)
	head := iconStr + " "
	if glyphs != "off" {
		head, nameStr = compactLogoPrefix(snap.ProviderID, glyphs, head, nameStr)
	}
	segs := make([]string, 0, 2)
	for _, e := range collectActiveResetEntries(snap, widget) {
		if len(segs) == 2 {
			break
		}
		pct, ok := compactPercentForReset(snap, e.key)
		if !ok {
			continue
		}
		pctStr := lipgloss.NewStyle().Bold(true).Foreground(
			usageGaugeColor(pct, compactWarnThresh, compactCritThresh),
		).Render(fmt.Sprintf("%d%%", int(math.Round(pct))))
		segs = append(segs, pctStr+dimStyle.Render("\u00b7"+formatHeaderDuration(e.dur)))
	}
	var line string
	if len(segs) == 0 {
		line = head + nameStr + " " + dimStyle.Render("no quota")
	} else {
		line = head + nameStr + "  " + strings.Join(segs, "  ")
	}
	if w > 0 {
		line = cropAnsiLine(line, 0, w)
	}
	return line
}

// compactPercentForReset resolves the metric behind a reset entry the same
// way the tile labels do (exact key, then the `_reset`-trimmed key) and
// returns its used percent, or false when unknown.
func compactPercentForReset(snap core.UsageSnapshot, key string) (float64, bool) {
	if met, ok := snap.Metrics[key]; ok {
		if pct := core.MetricUsedPercent(key, met); pct >= 0 {
			return pct, true
		}
	}
	if trimmed := strings.TrimSuffix(key, "_reset"); trimmed != key {
		if met, ok := snap.Metrics[trimmed]; ok {
			if pct := core.MetricUsedPercent(trimmed, met); pct >= 0 {
				return pct, true
			}
		}
	}
	return 0, false
}
