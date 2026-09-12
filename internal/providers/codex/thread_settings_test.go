package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/janekbaraniewski/openusage/internal/core"
)

func TestReadSessionUsageBreakdowns_ThreadSettingsApplied(t *testing.T) {
	tmp := t.TempDir()
	sessDir := filepath.Join(tmp, "sessions")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sessDir, "2026-09-12-test.jsonl")
	content := "{\"timestamp\":\"2026-09-12T14:00:00.000Z\",\"type\":\"session_meta\",\"payload\":{\"id\":\"test\",\"session_id\":\"test\",\"originator\":\"Codex Desktop\",\"source\":\"vscode\",\"model_provider\":\"openai\",\"base_instructions\":{\"provenance\":{\"type\":\"custom\"}}}}\n" +
		"{\"timestamp\":\"2026-09-12T14:00:01.000Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"thread_settings_applied\",\"thread_id\":\"t\",\"thread_settings\":{\"model\":\"gpt-6-astra\",\"model_provider_id\":\"openai\"}}}\n" +
		"{\"timestamp\":\"2026-09-12T14:00:02.000Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"info\":{\"total_token_usage\":{\"input_tokens\":1000000,\"cached_input_tokens\":0,\"output_tokens\":500000,\"reasoning_output_tokens\":0,\"total_tokens\":1500000},\"last_token_usage\":{\"input_tokens\":1000000,\"output_tokens\":500000,\"total_tokens\":1500000},\"model_context_window\":128000}}}\n" +
		"{\"timestamp\":\"2026-09-12T14:00:03.000Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"info\":{\"total_token_usage\":{\"input_tokens\":2000000,\"cached_input_tokens\":0,\"output_tokens\":1000000,\"reasoning_output_tokens\":0,\"total_tokens\":3000000},\"last_token_usage\":{\"input_tokens\":1000000,\"output_tokens\":500000,\"total_tokens\":1500000},\"model_context_window\":128000}}}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	p := New()
	acct := core.AccountConfig{ID: "test", Provider: "codex", RuntimeHints: map[string]string{"sessions_dir": sessDir, "config_dir": tmp}}
	snap, err := p.Fetch(context.Background(), acct)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if m, ok := snap.Metrics["model_gpt_6_astra_total_tokens"]; !ok || m.Used == nil || *m.Used != 3000000 {
		t.Fatalf("model_gpt_6_astra_total_tokens = %#v, want 3000000", m)
	}
	if _, ok := snap.Metrics["model_unknown_total_tokens"]; ok {
		t.Fatalf("should not have unknown, got %v", snap.Metrics["model_unknown_total_tokens"])
	}
}
