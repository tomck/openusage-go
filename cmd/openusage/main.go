package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/janekbaraniewski/openusage/internal/config"
	"github.com/janekbaraniewski/openusage/internal/core"
	"github.com/janekbaraniewski/openusage/internal/version"
	"github.com/spf13/cobra"
)

// resolveDashboardViewFlag validates a --view value against the known
// dashboard views. Comparison is case-insensitive; surrounding whitespace is
// ignored. Unknown values are an error (rather than a silent fallback) so a
// typo never boots the user into an unexpected view.
func resolveDashboardViewFlag(raw string, known []string) (string, error) {
	want := strings.ToLower(strings.TrimSpace(raw))
	for _, name := range known {
		if want == name {
			return name, nil
		}
	}
	return "", fmt.Errorf("unknown dashboard view %q (want one of: %s)", raw, strings.Join(known, ", "))
}

func main() {
	if core.DebugEnabled() {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(io.Discard)
	}

	// dashboardViewNames mirrors the TUI's view cycle (see
	// internal/tui/dashboard_views.go) so --view can name any of them.
	dashboardViewNames := []string{
		config.DashboardViewGrid,
		config.DashboardViewStacked,
		config.DashboardViewTabs,
		config.DashboardViewSplit,
		config.DashboardViewCompare,
		config.DashboardViewCompact,
	}
	var viewFlag string

	root := cobra.Command{
		Use:     "openusage",
		Short:   "OpenUsage is a terminal dashboard for monitoring AI coding tool usage and spend.",
		Version: version.Version,
		Run: func(_ *cobra.Command, _ []string) {
			// Loaded here rather than in main so an unreadable config only fails
			// the dashboard. Subcommands load their own config, and the ones that
			// do not need it (version, help, daemon install/uninstall) keep
			// working — including the ones you reach for to dig yourself out.
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				fmt.Fprintf(os.Stderr, "Config path: %s\n", config.ConfigPath())
				fmt.Fprintf(os.Stderr, "Move that file aside to start from defaults.\n")
				os.Exit(1)
			}
			if viewFlag != "" {
				view, err := resolveDashboardViewFlag(viewFlag, dashboardViewNames)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				cfg.Dashboard.View = view
			}

			runDashboard(cfg)
		},
	}
	root.Flags().StringVar(&viewFlag, "view", "",
		"dashboard view for this run: grid|stacked|tabs|split|compare|compact (overrides settings.json)")

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version.String())
		},
	})
	root.AddCommand(newTelemetryCommand())
	root.AddCommand(newIntegrationsCommand())
	root.AddCommand(newDetectCommand())
	root.AddCommand(newPricingCommand())
	root.AddCommand(newExportCommand())
	root.AddCommand(newHubCommand())
	root.AddCommand(newHubViewCommand())
	root.AddCommand(newAntigravityCommand())
	root.AddCommand(newStatuslineCommand())
	root.AddCommand(newTmuxCommand())
	for _, c := range newReportCommands() {
		root.AddCommand(c)
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
