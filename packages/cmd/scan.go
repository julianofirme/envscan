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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	Use:   "scan [repository]",
	Short: "Scan a Git repository for secrets",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoPath := args[0]
		config, err := loadConfig(configFile)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		scanRepository(repoPath, config)
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

func shouldIgnore(file string, config Config) bool {
	for _, ignoreFile := range config.Ignore.Files {
		if file == ignoreFile {
			return true
		}
	}
	for _, ignoreDir := range config.Ignore.Directories {
		if strings.HasPrefix(file, ignoreDir) {
			return true
		}
	}
	return false
}

func scanRepository(repoPath string, config Config) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		fmt.Printf("Error opening repository: %v\n", err)
		os.Exit(1)
	}

	iter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		fmt.Printf("Error getting logs: %v\n", err)
		os.Exit(1)
	}

	var matches []string

	err = iter.ForEach(func(c *object.Commit) error {
		files, err := c.Files()
		if err != nil {
			return err
		}

		err = files.ForEach(func(f *object.File) error {
			if shouldIgnore(f.Name, config) {
				return nil
			}

			content, err := f.Contents()
			if err != nil {
				return err
			}

			scanner := bufio.NewScanner(strings.NewReader(content))
			for scanner.Scan() {
				line := scanner.Text()
				for _, rule := range config.Rules {
					re := regexp.MustCompile(rule.Regex)
					if re.MatchString(line) {
						matches = append(matches, fmt.Sprintf("Match found in commit %s, file %s: %s", c.Hash, f.Name, line))
					}
				}
			}

			return scanner.Err()
		})

		return err
	})

	if err != nil {
		fmt.Printf("Error scanning commits: %v\n", err)
		os.Exit(1)
	}

	if len(matches) > 0 {
		fmt.Println("Potential secrets found:")
		for _, match := range matches {
			fmt.Println(match)
		}
		generateReport(matches, "json")

		if discordWebhookURL != "" {
			err := sendDiscordNotification(discordWebhookURL, "Secrets found in repository")
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
