package cli

import (
	"bytes"
	"fmt"
	"text/template"
)

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
