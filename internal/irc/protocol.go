package irc

import (
	"strings"

	"github.com/Ploos-AS/Golog/internal/core"
)

const maxIRCLine = 510 // bytes before CRLF

func sanitizeParam(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return strings.TrimSpace(s)
}

func sanitizeText(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func limitMessage(command, target, text string) string {
	command = strings.ToUpper(sanitizeParam(command))
	target = sanitizeParam(target)
	text = sanitizeText(text)
	prefix := command + " " + target + " :"
	budget := maxIRCLine - len(prefix)
	if budget < 0 {
		budget = 0
	}
	if len(text) > budget {
		text = text[:budget]
	}
	return prefix + text + "\r\n"
}

func limitPrivmsg(target, text string) string {
	return limitMessage("PRIVMSG", target, text)
}

// formatAction maps the constrained core action model to IRC wire commands.
// Unsupported commands are ignored rather than allowing arbitrary raw IRC.
func formatAction(a core.Action) (string, bool) {
	command := strings.ToUpper(sanitizeParam(a.Command))
	target := sanitizeParam(a.Target)
	arg := sanitizeParam(a.Arg)
	text := sanitizeText(a.Text)

	switch command {
	case "PRIVMSG", "NOTICE":
		if target == "" {
			return "", false
		}
		return limitMessage(command, target, text), true
	case "JOIN":
		if target == "" {
			return "", false
		}
		return "JOIN " + target + "\r\n", true
	case "PART":
		if target == "" {
			return "", false
		}
		if text == "" {
			return "PART " + target + "\r\n", true
		}
		return limitMessage("PART", target, text), true
	case "TOPIC":
		if target == "" {
			return "", false
		}
		return limitMessage("TOPIC", target, text), true
	case "MODE":
		if target == "" || arg == "" {
			return "", false
		}
		line := "MODE " + target + " " + arg
		if text != "" {
			line += " " + text
		}
		if len(line) > maxIRCLine {
			line = line[:maxIRCLine]
		}
		return line + "\r\n", true
	case "KICK":
		if target == "" || arg == "" {
			return "", false
		}
		prefix := "KICK " + target + " " + arg
		if text == "" {
			if len(prefix) > maxIRCLine {
				prefix = prefix[:maxIRCLine]
			}
			return prefix + "\r\n", true
		}
		prefix += " :"
		budget := maxIRCLine - len(prefix)
		if budget < 0 {
			budget = 0
		}
		if len(text) > budget {
			text = text[:budget]
		}
		return prefix + text + "\r\n", true
	default:
		return "", false
	}
}

func isChannelTarget(target string) bool {
	if target == "" {
		return false
	}
	switch target[0] {
	case '#', '&', '+', '!':
		return true
	default:
		return false
	}
}
