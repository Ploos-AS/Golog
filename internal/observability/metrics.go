package observability

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Metrics contains lightweight runtime counters/gauges without external dependencies.
type Metrics struct {
	started time.Time

	connected atomic.Bool
	ready     atomic.Bool

	events         atomic.Uint64
	actions        atomic.Uint64
	reconnects     atomic.Uint64
	reloads        atomic.Uint64
	reloadFailures atomic.Uint64
	stateErrors    atomic.Uint64

	queueDepth    atomic.Int64
	queueCapacity atomic.Int64
}

func NewMetrics() *Metrics { return &Metrics{started: time.Now()} }

func (m *Metrics) SetConnected(v bool) { m.connected.Store(v) }
func (m *Metrics) SetReady(v bool)     { m.ready.Store(v) }
func (m *Metrics) Connected() bool     { return m.connected.Load() }
func (m *Metrics) Ready() bool         { return m.ready.Load() }

func (m *Metrics) IncEvents()         { m.events.Add(1) }
func (m *Metrics) IncActions()        { m.actions.Add(1) }
func (m *Metrics) IncReconnects()     { m.reconnects.Add(1) }
func (m *Metrics) IncReloads()        { m.reloads.Add(1) }
func (m *Metrics) IncReloadFailures() { m.reloadFailures.Add(1) }
func (m *Metrics) IncStateErrors()    { m.stateErrors.Add(1) }

func (m *Metrics) SetQueue(depth, capacity int) {
	m.queueDepth.Store(int64(depth))
	m.queueCapacity.Store(int64(capacity))
}

func boolFloat(v bool) int {
	if v {
		return 1
	}
	return 0
}

// Prometheus returns metrics in the Prometheus text exposition format.
func (m *Metrics) Prometheus() string {
	return fmt.Sprintf(`# TYPE golog_up gauge
golog_up 1
# TYPE golog_ready gauge
golog_ready %d
# TYPE golog_irc_connected gauge
golog_irc_connected %d
# TYPE golog_events_total counter
golog_events_total %d
# TYPE golog_actions_total counter
golog_actions_total %d
# TYPE golog_reconnects_total counter
golog_reconnects_total %d
# TYPE golog_reloads_total counter
golog_reloads_total %d
# TYPE golog_reload_failures_total counter
golog_reload_failures_total %d
# TYPE golog_state_errors_total counter
golog_state_errors_total %d
# TYPE golog_event_queue_depth gauge
golog_event_queue_depth %d
# TYPE golog_event_queue_capacity gauge
golog_event_queue_capacity %d
# TYPE golog_uptime_seconds gauge
golog_uptime_seconds %.0f
`,
		boolFloat(m.Ready()), boolFloat(m.Connected()),
		m.events.Load(), m.actions.Load(), m.reconnects.Load(), m.reloads.Load(),
		m.reloadFailures.Load(), m.stateErrors.Load(), m.queueDepth.Load(),
		m.queueCapacity.Load(), time.Since(m.started).Seconds())
}
