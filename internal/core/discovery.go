package core

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Ploos-AS/Golog/internal/rules"
)

var workerRegistries sync.Map // map[*Worker]*rules.Registry

// SetRegistry updates the discovery registry associated with this worker.
// Registry objects are immutable after construction, so pointer replacement is
// safe while the worker is running.
func (w *Worker) SetRegistry(registry *rules.Registry) {
	if registry == nil {
		registry = rules.EmptyRegistry()
	}
	workerRegistries.Store(w, registry)
}

func (w *Worker) registry() *rules.Registry {
	if value, ok := workerRegistries.Load(w); ok {
		if registry, ok := value.(*rules.Registry); ok && registry != nil {
			return registry
		}
	}
	return rules.EmptyRegistry()
}

func (w *Worker) handleDiscovery(ctx context.Context, ev Event) (bool, error) {
	fields := strings.Fields(ev.Text)
	if len(fields) == 0 {
		return false, nil
	}

	command := strings.ToLower(fields[0])
	registry := w.registry()
	var text string

	switch command {
	case "!help":
		commands := registry.Commands()
		base := []string{"!help", "!packs", "!experts", "!capabilities"}
		commands = append(base, commands...)
		text = "commands: " + strings.Join(uniqueSorted(commands), ", ")
	case "!packs":
		ids := registry.IDs()
		if len(fields) > 1 {
			manifest, ok := registry.Lookup(fields[1])
			if !ok {
				text = "pack not found: " + fields[1]
			} else {
				text = fmt.Sprintf("pack %s %s: %s", manifest.ID, manifest.Version, manifest.Description)
				if len(manifest.Roles) > 0 {
					text += " | roles=" + strings.Join(manifest.Roles, ",")
				}
				if len(manifest.Commands) > 0 {
					text += " | commands=" + strings.Join(manifest.Commands, ",")
				}
				if len(manifest.Capabilities) > 0 {
					text += " | capabilities=" + strings.Join(manifest.Capabilities, ",")
				}
			}
		} else if len(ids) == 0 {
			text = "packs: none"
		} else {
			text = "packs: " + strings.Join(ids, ", ")
		}
	case "!experts":
		roles := registry.Roles()
		if len(roles) == 0 {
			text = "experts: none"
		} else {
			text = "experts: " + strings.Join(roles, ", ")
		}
	case "!capabilities":
		capabilities := registry.Capabilities()
		if len(capabilities) == 0 {
			text = "capabilities: none"
		} else {
			text = "capabilities: " + strings.Join(capabilities, ", ")
		}
	default:
		return false, nil
	}

	target := ev.Target
	if target == "" || strings.EqualFold(target, "Golog") {
		target = ev.Nick
	}
	return true, w.emit(ctx, Action{Command: "PRIVMSG", Target: target, Text: text})
}

func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	// registry lists are sorted already; sort again because base commands were appended.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
