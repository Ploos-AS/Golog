package irc

import "strings"

type Message struct {
	Tags    map[string]string
	Prefix  string
	Command string
	Params  []string
	Trail   string
}

func Parse(line string) Message {
	line = strings.TrimRight(line, "\r\n")
	m := Message{Tags: map[string]string{}}
	if strings.HasPrefix(line, "@") {
		if i := strings.IndexByte(line, ' '); i >= 0 {
			for _, raw := range strings.Split(line[1:i], ";") {
				key, value, _ := strings.Cut(raw, "=")
				if key != "" {
					m.Tags[key] = unescapeTag(value)
				}
			}
			line = strings.TrimLeft(line[i+1:], " ")
		}
	}
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

func unescapeTag(s string) string {
	r := strings.NewReplacer(`\:`, ";", `\s`, " ", `\\`, `\`, `\r`, "\r", `\n`, "\n")
	return r.Replace(s)
}

func NickFromPrefix(prefix string) string {
	if i := strings.IndexByte(prefix, '!'); i >= 0 {
		return prefix[:i]
	}
	return prefix
}
