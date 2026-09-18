// Package info registers the Magento 2 handler for the "info" command.
// It runs magento-info.php inside the php container with --format=json,
// parses what it answers and hands the report blocks back to general/info,
// which renders them in whatever format the user asked for.
package info

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/faradey/madock/v4/src/helper/docker"
	inforeg "github.com/faradey/madock/v4/src/info"
)

type Handler struct{}

// report mirrors what scripts/php/magento-info.php prints as JSON.
type report struct {
	Platform struct {
		Edition    string  `json:"edition"`
		Constraint *string `json:"constraint"`
		Installed  *string `json:"installed"`
	} `json:"platform"`
	Modules []struct {
		Name    string  `json:"name"`
		Package *string `json:"package"`
		Version *string `json:"version"`
		Latest  *string `json:"latest"`
		Enabled bool    `json:"enabled"`
	} `json:"modules"`
	Warnings []string `json:"warnings"`
}

func (h *Handler) Collect(ctx *inforeg.InfoContext) ([]inforeg.Block, error) {
	containerName := docker.GetContainerName(ctx.ProjectConf, ctx.ProjectName, ctx.Service)
	command := []string{"php", "/var/www/scripts/php/magento-info.php", ctx.ProjectConf["workdir"], "--format=json"}

	// Not interactive: a TTY would turn the JSON into CRLF lines and echo
	// nothing useful. stderr stays on the terminal so a PHP fatal is seen.
	cmd, err := docker.PrepareContainerExec(containerName, "", false, command...)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	execErr := cmd.Run()
	docker.NotifyExecDone(containerName, command, execErr)
	if execErr != nil {
		return nil, fmt.Errorf("magento-info.php failed: %w", execErr)
	}

	var r report
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		return nil, fmt.Errorf("magento-info.php printed something that is not JSON: %w\n%s", err, out.String())
	}

	blocks := []inforeg.Block{{
		Key:   "magento",
		Title: "Magento",
		Items: []inforeg.Item{
			{Key: "edition", Value: r.Platform.Edition},
			{Key: "constraint", Value: orDash(r.Platform.Constraint)},
			{Key: "installed", Value: orDash(r.Platform.Installed)},
		},
	}}

	rows := make([][]string, 0, len(r.Modules))
	for _, m := range r.Modules {
		status := "enabled"
		if !m.Enabled {
			status = "disabled"
		}
		rows = append(rows, []string{m.Name, orDash(m.Package), orDash(m.Version), orDash(m.Latest), status})
	}
	blocks = append(blocks, inforeg.Block{
		Key:     "modules",
		Title:   "Third-party modules",
		RowName: "module",
		Columns: []inforeg.Column{
			{Key: "name", Title: "Name"},
			{Key: "package", Title: "Package"},
			{Key: "version", Title: "Version"},
			{Key: "latest", Title: "Latest"},
			{Key: "status", Title: "Status"},
		},
		Rows: rows,
	})

	if len(r.Warnings) > 0 {
		blocks = append(blocks, inforeg.Block{Key: "warnings", Title: "Warning", Lines: r.Warnings})
	}

	return blocks, nil
}

// orDash prints "-" for what the script could not establish, so a missing
// version reads as missing in every format rather than as an empty cell.
func orDash(s *string) string {
	if s == nil || *s == "" {
		return "-"
	}
	return *s
}

func init() {
	inforeg.Register("magento2", &Handler{})
}
