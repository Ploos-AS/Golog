package pbmp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type State struct {
	Nick      string
	Network   string
	Connected bool
}

type request struct {
	PBMP   int             `json:"pbmp"`
	Type   string          `json:"type"`
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type response struct {
	PBMP   int         `json:"pbmp"`
	Type   string      `json:"type"`
	ID     string      `json:"id"`
	OK     bool        `json:"ok"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

func Serve(path string, state State) error {
	if path == "" {
		return fmt.Errorf("PBMP socket path is empty")
	}
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	defer func() {
		_ = ln.Close()
		_ = os.Remove(path)
	}()
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go handle(conn, state)
	}
}

func handle(conn net.Conn, state State) {
	defer conn.Close()
	s := bufio.NewScanner(conn)
	if !s.Scan() {
		return
	}
	var req request
	if err := json.Unmarshal(s.Bytes(), &req); err != nil || req.PBMP != 1 || req.Type != "request" {
		return
	}
	res := response{PBMP: 1, Type: "response", ID: req.ID, OK: true}
	switch req.Method {
	case "pbmp.info":
		res.Result = map[string]interface{}{
			"version": 1,
			"implementation": map[string]string{"name": "golog", "version": "0.1.0"},
		}
	case "capabilities.list":
		res.Result = map[string]interface{}{"capabilities": []string{"pbmp.info", "capabilities.list", "bot.info", "networks.list"}}
	case "bot.info":
		stateName := "stopped"
		if state.Connected {
			stateName = "running"
		}
		res.Result = map[string]interface{}{"bot": map[string]interface{}{
			"id": state.Nick,
			"implementation": map[string]string{"name": "golog", "version": "0.1.0"},
			"state": stateName,
		}}
	case "networks.list":
		stateName := "disconnected"
		if state.Connected {
			stateName = "connected"
		}
		res.Result = map[string]interface{}{"networks": []map[string]string{{"id": state.Network, "name": state.Network, "state": stateName}}}
	default:
		res.OK = false
		res.Error = map[string]string{"code": "not_supported", "message": "method not supported"}
	}
	enc := json.NewEncoder(conn)
	_ = enc.Encode(res)
}
