package irc

import "strings"

type Message struct {
	Prefix  string
	Command string
	Params  []string
	Trail   string
}

func Parse(line string) Message {
	line = strings.TrimRight(line, "\r\n")
	m := Message{}
	if strings.HasPrefix(line, ":") {
		if i := strings.IndexByte(line, ' '); i >= 0 {
			m.Prefix = line[1:i]
			line = strings.TrimLeft(line[i+1:], " ")
		}
	}
	if i := strings.Index(line, " :"); i >= 0 {
		m.Trail = line[i+2:]
		line = line[:i]
	}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return m
	}
	m.Command = strings.ToUpper(parts[0])
	if len(parts) > 1 {
		m.Params = parts[1:]
	}
	return m
}

func NickFromPrefix(prefix string) string {
	if i := strings.IndexByte(prefix, '!'); i >= 0 {
		return prefix[:i]
	}
	return prefix
}
