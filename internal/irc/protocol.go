package irc

import "strings"

const maxIRCLine = 510 // bytes before CRLF

func sanitizeParam(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func sanitizeText(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func limitPrivmsg(target, text string) string {
	target = sanitizeParam(target)
	text = sanitizeText(text)
	prefix := "PRIVMSG " + target + " :"
	budget := maxIRCLine - len(prefix)
	if budget < 0 {
		budget = 0
	}
	if len(text) > budget {
		text = text[:budget]
	}
	return prefix + text + "\r\n"
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
