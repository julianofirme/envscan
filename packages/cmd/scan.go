package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"envscan/packages/config"
	"envscan/packages/notify"
	"envscan/packages/report"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var configFile string
var discordWebhookURL string

var scanCmd = &cobra.Command{
	Use:   "run [directory]",
	Short: "Run the scan in a directory",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dirPath := args[0]
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		scanDirectory(dirPath, cfg)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringVarP(&configFile, "config", "c", "rules.toml", "Path to the configuration file")
	scanCmd.Flags().StringVarP(&discordWebhookURL, "discord-webhook", "d", "", "Discord Webhook URL for notifications")
}

func scanDirectory(dirPath string, cfg config.Config) {
	var matches []string
	var fileCount int

	// Count total files for progress bar
	err := filepath.WalkDir(dirPath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(info.Name(), ".env") {
			fileCount++
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error counting files: %v\n", err)
		os.Exit(1)
	}

	// Initialize progress bar
	bar := progressbar.Default(int64(fileCount), "Scanning")

	// Scan files with progress bar
	err = filepath.WalkDir(dirPath, func(path string, info os.DirEntry, err error) error {
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
			for _, rule := range cfg.Rules {
				re := regexp.MustCompile(rule.Regex)
				if re.MatchString(line) {
					matches = append(matches, fmt.Sprintf("Potential secret found in file %s: %s", path, line))
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}

		bar.Add(1)
		return nil
	})

	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println() // Ensure newline after progress bar

	if len(matches) > 0 {
		fmt.Println("Potential secrets found:")
		for _, match := range matches {
			fmt.Println(match)
		}
		report.GenerateReport(matches, "json")

		if discordWebhookURL != "" {
			err := notify.SendDiscordNotification(discordWebhookURL, "Secrets found in directory")
			if err != nil {
				fmt.Printf("Error sending Discord notification: %v\n", err)
			}
		}

		os.Exit(1)
	} else {
		fmt.Println("No secrets found")
	}
}
