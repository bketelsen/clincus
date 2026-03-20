package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const serviceUnitName = "clincus.service"

func serviceUnitPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}
	return filepath.Join(homeDir, ".config", "systemd", "user", serviceUnitName)
}

var serviceUnitTemplate = `[Unit]
Description=Clincus Web Dashboard
After=network.target

[Service]
Type=simple
ExecStart={{.BinaryPath}} serve
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

type unitTemplateData struct {
	BinaryPath string
}

func renderUnitFile(binaryPath string) (string, error) {
	tmpl, err := template.New("unit").Parse(serviceUnitTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse unit template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, unitTemplateData{BinaryPath: binaryPath}); err != nil {
		return "", fmt.Errorf("failed to render unit template: %w", err)
	}

	return buf.String(), nil
}
