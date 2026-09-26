package ichiban

import (
	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
	"github.com/ichiban/prolog"
)

// Engine embeds the default Ichiban Prolog interpreter.
type Engine struct {
	p *prolog.Interpreter
}

// New returns a Golog Prolog engine backed by ichiban/prolog.
func New() *Engine {
	return &Engine{p: prolog.New(nil, nil)}
}

// Load adds Prolog source to the interpreter.
func (e *Engine) Load(source string) error {
	return e.p.Exec(source)
}

// Ask evaluates a Prolog goal and reports whether at least one solution exists.
func (e *Engine) Ask(query string, args ...any) (bool, error) {
	solutions, err := e.p.Query(query, args...)
	if err != nil {
		return false, err
	}
	defer solutions.Close()

	if solutions.Next() {
		return true, nil
	}
	if err := solutions.Err(); err != nil {
		return false, err
	}
	return false, nil
}

// QueryReply evaluates a goal whose result variable is named Reply.
func (e *Engine) QueryReply(query string, args ...any) (string, bool, error) {
	solutions, err := e.p.Query(query, args...)
	if err != nil {
		return "", false, err
	}
	defer solutions.Close()

	if !solutions.Next() {
		if err := solutions.Err(); err != nil {
			return "", false, err
		}
		return "", false, nil
	}

	var result struct {
		Reply string
	}
	if err := solutions.Scan(&result); err != nil {
		return "", false, err
	}
	return result.Reply, true, nil
}

// QueryActions evaluates a goal that returns one structured action per solution.
func (e *Engine) QueryActions(query string, args ...any) ([]gprolog.RuleAction, error) {
	solutions, err := e.p.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer solutions.Close()

	var out []gprolog.RuleAction
	for solutions.Next() {
		var result struct {
			Command string
			Target  string
			Arg     string
			Text    string
		}
		if err := solutions.Scan(&result); err != nil {
			return nil, err
		}
		out = append(out, gprolog.RuleAction{
			Command: result.Command,
			Target:  result.Target,
			Arg:     result.Arg,
			Text:    result.Text,
		})
	}
	if err := solutions.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
