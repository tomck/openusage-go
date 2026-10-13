package muse_code

import (
	"testing"

	"github.com/janekbaraniewski/openusage/internal/core"
)

func TestApplyPlanNameOverride_WinsOverTier(t *testing.T) {
	acct := core.AccountConfig{ID: "muse-code", Provider: "muse_code"}
	acct.SetPath(PlanNamePathKey, "Everyday Usage")
	snap := core.NewUsageSnapshot("muse_code", "muse-code")
	snap.Raw["plan_name"] = "27681393394859588"

	applyPlanNameOverride(acct, &snap)

	if got := snap.Raw["plan_name"]; got != "Everyday Usage" {
		t.Fatalf("plan_name = %q, want user-attested override", got)
	}
}

func TestApplyPlanNameOverride_EmptyLeavesTier(t *testing.T) {
	acct := core.AccountConfig{ID: "muse-code", Provider: "muse_code"}
	snap := core.NewUsageSnapshot("muse_code", "muse-code")
	snap.Raw["plan_name"] = "27681393394859588"

	applyPlanNameOverride(acct, &snap)

	if got := snap.Raw["plan_name"]; got != "27681393394859588" {
		t.Fatalf("plan_name = %q, want tier passthrough untouched", got)
	}
}

func TestApplyPlanNameOverride_ServerPlanWinsOverOverride(t *testing.T) {
	acct := core.AccountConfig{ID: "muse-code", Provider: "muse_code"}
	acct.SetPath(PlanNamePathKey, "High Usage")
	snap := core.NewUsageSnapshot("muse_code", "muse-code")
	snap.Raw["plan_name"] = "Everyday Usage"
	snap.SetAttribute(museQuotaServerPlanAttr, "Everyday Usage")

	applyPlanNameOverride(acct, &snap)

	if got := snap.Raw["plan_name"]; got != "Everyday Usage" {
		t.Fatalf("plan_name = %q, want server display name, not stale override", got)
	}
}

func TestApplySubscriptionUsage_MarksServerPlan(t *testing.T) {
	snap := core.NewUsageSnapshot("muse_code", "muse-code")
	sub := &subscriptionUsage{Tier: "27681393394859588", PlanName: "Everyday Usage"}
	sub.Weekly.UsedPercent = 10
	sub.Weekly.ResetsAt = 1789948800
	applySubscriptionUsage(&snap, sub)

	if got := snap.Attributes[museQuotaServerPlanAttr]; got != "Everyday Usage" {
		t.Fatalf("server plan attr = %q, want Everyday Usage", got)
	}

	// A stale saved label must not overwrite it (the reported downgrade bug).
	acct := core.AccountConfig{ID: "muse-code", Provider: "muse_code"}
	acct.SetPath(PlanNamePathKey, "High Usage")
	applyPlanNameOverride(acct, &snap)
	if got := snap.Raw["plan_name"]; got != "Everyday Usage" {
		t.Fatalf("plan_name = %q, want server name after override", got)
	}
}
