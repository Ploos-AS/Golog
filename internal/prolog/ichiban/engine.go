package ichiban

import "github.com/ichiban/prolog"

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

// QueryString evaluates a goal and returns one named string variable from
// the first solution.
func (e *Engine) QueryString(query, variable string, args ...any) (string, bool, error) {
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

	values := map[string]any{}
	if err := solutions.Scan(&values); err != nil {
		return "", false, err
	}
	v, ok := values[variable]
	if !ok {
		return "", false, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", false, nil
	}
	return s, true, nil
}
