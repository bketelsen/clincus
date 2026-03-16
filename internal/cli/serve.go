package cli

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"

	"github.com/mensfeld/code-on-incus/internal/server"
	"github.com/mensfeld/code-on-incus/webui"
	"github.com/spf13/cobra"
)

var servePort int
var serveOpen bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web dashboard",
	RunE:  serveCommand,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 0, "Port to listen on (default from config or 3000)")
	serveCmd.Flags().BoolVar(&serveOpen, "open", false, "Open browser after starting")
}

func serveCommand(cmd *cobra.Command, args []string) error {
	port := cfg.Dashboard.Port
	if port == 0 {
		port = 3000
	}
	if servePort != 0 {
		port = servePort
	}

	assets, err := fs.Sub(webui.Dist, "dist")
	if err != nil {
		return fmt.Errorf("failed to load web assets: %w", err)
	}

	srv := server.New(server.Options{
		Port:      port,
		Assets:    assets,
		AppConfig: cfg,
	})
	srv.Start()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("Dashboard running at http://%s", addr)

	if serveOpen {
		go func() {
			exec.Command("xdg-open", fmt.Sprintf("http://%s", addr)).Start()
		}()
	}

	return http.ListenAndServe(addr, srv.Handler())
}
