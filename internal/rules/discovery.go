package rules

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// DiscoverySource returns core discovery clauses generated from the active
// manifest registry. The source is loaded atomically together with all pack
// files, so discovery always describes the same ruleset that is executing.
func DiscoverySource(registry *Registry) string {
	if registry == nil {
		registry = EmptyRegistry()
	}

	commands := append([]string{"!help", "!packs", "!experts", "!capabilities"}, registry.Commands()...)
	commands = uniqueStrings(commands)

	var source strings.Builder
	source.WriteString("\n% --- Golog M1.3 generated discovery rules ---\n")
	writeReply(&source, "!help", "commands: "+strings.Join(commands, ", "))

	ids := registry.IDs()
	if len(ids) == 0 {
		writeReply(&source, "!packs", "packs: none")
	} else {
		writeReply(&source, "!packs", "packs: "+strings.Join(ids, ", "))
	}

	roles := registry.Roles()
	if len(roles) == 0 {
		writeReply(&source, "!experts", "experts: none")
	} else {
		writeReply(&source, "!experts", "experts: "+strings.Join(roles, ", "))
	}

	capabilities := registry.Capabilities()
	if len(capabilities) == 0 {
		writeReply(&source, "!capabilities", "capabilities: none")
	} else {
		writeReply(&source, "!capabilities", "capabilities: "+strings.Join(capabilities, ", "))
	}

	for _, manifest := range registry.Packs() {
		detail := fmt.Sprintf("pack %s %s: %s", manifest.ID, manifest.Version, manifest.Description)
		if len(manifest.Roles) > 0 {
			detail += " | roles=" + strings.Join(manifest.Roles, ",")
		}
		if len(manifest.Commands) > 0 {
			detail += " | commands=" + strings.Join(manifest.Commands, ",")
		}
		if len(manifest.Capabilities) > 0 {
			detail += " | capabilities=" + strings.Join(manifest.Capabilities, ",")
		}
		writeReply(&source, "!packs "+manifest.ID, detail)
	}
	return source.String()
}

func writeReply(source *strings.Builder, command, reply string) {
	fmt.Fprintf(source, "on_privmsg(_Nick, _Target, %s, %s).\n", strconv.Quote(command), strconv.Quote(reply))
}

func uniqueStrings(values []string) []string {
	set := map[string]struct{}{}
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
	sort.Strings(out)
	return out
}
