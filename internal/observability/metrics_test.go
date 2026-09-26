package observability

import (
	"strings"
	"testing"
)

func TestPrometheusMetrics(t *testing.T) {
	m := NewMetrics()
	m.SetConnected(true)
	m.SetReady(true)
	m.IncEvents()
	m.IncActions()
	m.IncReconnects()
	m.IncReloads()
	m.IncReloadFailures()
	m.IncStateErrors()
	m.SetQueue(3, 64)

	got := m.Prometheus()
	for _, want := range []string{
		"golog_up 1",
		"golog_ready 1",
		"golog_irc_connected 1",
		"golog_events_total 1",
		"golog_actions_total 1",
		"golog_reconnects_total 1",
		"golog_reloads_total 1",
		"golog_reload_failures_total 1",
		"golog_state_errors_total 1",
		"golog_event_queue_depth 3",
		"golog_event_queue_capacity 64",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("metrics missing %q:\n%s", want, got)
		}
	}
}
