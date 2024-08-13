package report

import (
	"encoding/json"
	"os"
)

func GenerateReport(matches []string, format string) error {
	if format != "json" {
		return nil
	}

	report := map[string][]string{
		"secrets": matches,
	}

	file, err := os.Create("secrets-report.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return err
	}

	return nil
}
