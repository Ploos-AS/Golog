package pbmp

import (
	"bufio"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func call(t *testing.T, socket, method string) map[string]interface{} {
	t.Helper()
	conn, err := net.Dial("unix", socket)
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	_, _ = conn.Write([]byte(`{"pbmp":1,"type":"request","id":"1","method":"` + method + `","params":{}}` + "\n"))
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil { t.Fatal(err) }
	var out map[string]interface{}
	if err := json.Unmarshal(line, &out); err != nil { t.Fatal(err) }
	return out
}

func TestRequiredMethods(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "golog.pbmp.sock")
	go func() { _ = Serve(socket, State{Nick: "golog", Network: "irc.example"}) }()
	for i := 0; i < 50; i++ {
		if c, err := net.Dial("unix", socket); err == nil { _ = c.Close(); break }
		time.Sleep(10 * time.Millisecond)
	}
	for _, method := range []string{"pbmp.info", "capabilities.list", "bot.info", "networks.list"} {
		out := call(t, socket, method)
		if ok, _ := out["ok"].(bool); !ok { t.Fatalf("%s failed: %#v", method, out) }
	}
}
