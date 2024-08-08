package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func GenerateReport(matches []string, format string) error {
	report := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"matches":   matches,
	}

	var output []byte
	var err error
	switch format {
	case "json":
		output, err = json.MarshalIndent(report, "", "  ")
	case "html":
		// TODO: implement HTML report
	case "csv":
		// TODO: implement CSV report
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join("reports", fmt.Sprintf("report_%d.%s", time.Now().Unix(), format)), output, 0644)
}
