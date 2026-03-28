package server

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
	Code int    `json:"code,omitempty"`
	Msg  string `json:"message,omitempty"`
}

type Bridge struct {
	cmd  *exec.Cmd
	ptmx *os.File
	ws   *websocket.Conn
	once sync.Once
}

func NewBridge(ws *websocket.Conn, containerName string, execArgs []string) (*Bridge, error) {
	incusArgs := execArgs

	var cmd *exec.Cmd
	if runtime.GOOS == "linux" {
		fullCmd := "incus"
		for _, a := range incusArgs {
			fullCmd += " " + shellQuote(a)
		}
		cmd = exec.Command("sg", "incus-admin", "-c", fullCmd) //nolint:gosec // args are shell-quoted; sg requires string
	} else {
		cmd = exec.Command("incus", incusArgs...) //nolint:gosec // args are controlled incus parameters
	}
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return &Bridge{cmd: cmd, ptmx: ptmx, ws: ws}, nil
}

func (b *Bridge) Run() {
	var wg sync.WaitGroup
	wg.Add(2)

	// PTY -> WebSocket
	go func() {
		defer wg.Done()
		buf := make([]byte, 16384)
		for {
			n, err := b.ptmx.Read(buf)
			if n > 0 {
				msg := WSMessage{Type: "output", Data: string(buf[:n])}
				if werr := b.ws.WriteJSON(msg); werr != nil {
					b.Close()
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket -> PTY
	go func() {
		defer wg.Done()
		for {
			_, raw, err := b.ws.ReadMessage()
			if err != nil {
				b.Close()
				return
			}
			var msg WSMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			switch msg.Type {
			case "input":
				//nolint:errcheck // write to PTY; error handled via next read failure
				_, _ = b.ptmx.Write([]byte(msg.Data))
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					//nolint:errcheck,gosec // resize failure non-fatal; int->uint16 bounded by terminal dimensions
					_ = pty.Setsize(b.ptmx, &pty.Winsize{
						Cols: uint16(msg.Cols),
						Rows: uint16(msg.Rows),
					})
				}
			}
		}
	}()

	exitCode := 0
	if err := b.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	//nolint:errcheck // best-effort exit notification to client
	_ = b.ws.WriteJSON(WSMessage{Type: "exit", Code: exitCode})
	_ = b.ws.Close()
	wg.Wait()
}

func (b *Bridge) Close() {
	b.once.Do(func() {
		if b.cmd.Process != nil {
			//nolint:errcheck // best-effort process kill
			_ = b.cmd.Process.Kill()
		}
		b.ptmx.Close()
	})
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
