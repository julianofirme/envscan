package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/spf13/cobra"
)

type Rule struct {
	Description string   `toml:"description"`
	ID          string   `toml:"id"`
	Regex       string   `toml:"regex"`
	SecretGroup int      `toml:"secretGroup"`
	Keywords    []string `toml:"keywords"`
}

type Config struct {
	Rules  []Rule `toml:"rules"`
	Ignore Ignore `toml:"ignore"`
}

type Ignore struct {
	Files       []string `toml:"files"`
	Directories []string `toml:"directories"`
}

var configFile string
var discordWebhookURL string

var scanCmd = &cobra.Command{
	Use:   "run [repository]",
	Short: "Run the scan in a Git repository for secrets",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoPath := args[0]
		config, err := loadConfig(configFile)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		scanDirectory(repoPath, config)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringVarP(&configFile, "config", "c", "rules.toml", "Path to the configuration file")
	scanCmd.Flags().StringVarP(&discordWebhookURL, "discord-webhook", "d", "", "Discord Webhook URL for notifications")
}

func loadConfig(filePath string) (Config, error) {
	var config Config
	content, err := os.ReadFile(filePath)
	if err != nil {
		return config, err
	}
	err = toml.Unmarshal(content, &config)
	return config, err
}

func scanDirectory(dirPath string, config Config) {
	var matches []string

	err := filepath.WalkDir(dirPath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.HasSuffix(info.Name(), ".env") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			for _, rule := range config.Rules {
				re := regexp.MustCompile(rule.Regex)
				if re.MatchString(line) {
					matches = append(matches, fmt.Sprintf("Match found in file %s: %s", path, line))
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	if len(matches) > 0 {
		fmt.Println("Potential secrets found:")
		for _, match := range matches {
			fmt.Println(match)
		}
		generateReport(matches, "json")

		if discordWebhookURL != "" {
			err := sendDiscordNotification(discordWebhookURL, "Secrets found in directory")
			if err != nil {
				fmt.Printf("Error sending Discord notification: %v\n", err)
			}
		}

		os.Exit(1)
	} else {
		fmt.Println("No secrets found")
	}
}

func generateReport(matches []string, format string) error {
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
		return fmt.Errorf("formato desconhecido: %s", format)
	}
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join("reports", fmt.Sprintf("report_%d.%s", time.Now().Unix(), format)), output, 0644)
}

func sendDiscordNotification(webhookURL, content string) error {
	message := map[string]string{"content": content}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(messageBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to send notification: %s", resp.Status)
	}

	return nil
}
