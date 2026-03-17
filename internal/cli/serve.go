package cli

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	cfgpkg "github.com/bketelsen/clincus/internal/config"
	"github.com/bketelsen/clincus/internal/server"
	"github.com/bketelsen/clincus/webui"
	"github.com/spf13/cobra"
)

var (
	servePort int
	serveOpen bool
)

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

	// restartCh is signaled when a config reload changes the dashboard port.
	restartCh := make(chan int, 1)

	srv := server.New(server.Options{
		Port:      port,
		Assets:    assets,
		AppConfig: cfg,
	})
	srv.Start()

	// Set up config hot-reload via ConfigManager.
	configMgr, err := cfgpkg.NewConfigManager(func(oldCfg, newCfg *cfgpkg.Config) {
		oldPort := srv.UpdateConfig(newCfg)

		// AC1/AC5: Broadcast config.reloaded event to all connected WebSocket
		// clients. This only fires on successful reloads — failed reloads never
		// reach the onChange callback (see ConfigManager.reload).
		srv.BroadcastConfigReloaded()

		newPort := newCfg.Dashboard.Port
		if newPort == 0 {
			newPort = 3000
		}
		if oldPort != newPort {
			log.Printf("Dashboard port changed %d -> %d, cycling listener", oldPort, newPort)
			// Non-blocking send; if a restart is already pending it will pick up the latest port.
			select {
			case restartCh <- newPort:
			default:
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}
	defer configMgr.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("Dashboard running at http://%s", addr)

	if serveOpen {
		go func() {
			//nolint:errcheck // fire-and-forget browser open; failure is non-fatal
			_ = exec.Command("xdg-open", fmt.Sprintf("http://%s", addr)).Start()
		}()
	}

	// Listener loop: supports graceful restart when dashboard port changes (AC4).
	return listenLoop(srv.Handler(), port, restartCh)
}

// listenLoop runs the HTTP server and restarts the listener when a new port
// arrives on restartCh. Active connections are gracefully drained on shutdown.
func listenLoop(handler http.Handler, initialPort int, restartCh <-chan int) error {
	var (
		mu         sync.Mutex
		currentSrv *http.Server
	)

	startServer := func(port int) *http.Server {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		httpSrv := &http.Server{
			Addr:    addr,
			Handler: handler,
		}

		mu.Lock()
		currentSrv = httpSrv
		mu.Unlock()

		return httpSrv
	}

	// Start initial server.
	httpSrv := startServer(initialPort)

	// Listen for port-change signals in a separate goroutine.
	go func() {
		for newPort := range restartCh {
			mu.Lock()
			old := currentSrv
			mu.Unlock()

			if old != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := old.Shutdown(ctx); err != nil {
					log.Printf("Graceful shutdown error: %v", err)
				}
				cancel()
			}

			log.Printf("Dashboard now listening at http://127.0.0.1:%d", newPort)
			newSrv := startServer(newPort)
			go func(s *http.Server) {
				if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("Server error: %v", err)
				}
			}(newSrv)
		}
	}()

	// Block on the initial server.
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	// If we reach here it means the initial server was shut down due to a port
	// change. Block indefinitely (the restart goroutine handles new servers).
	select {}
}
