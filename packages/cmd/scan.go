package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

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

func trackTime(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Printf("%s took %s\n", name, elapsed)
}

func parseGitignore(dirPath string) ([]string, error) {
	var patterns []string
	gitignorePath := filepath.Join(dirPath, ".gitignore")

	file, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return patterns, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

func shouldIgnore(path string, dirPath string, patterns []string) bool {
	relativePath, err := filepath.Rel(dirPath, path)
	if err != nil {
		fmt.Printf("Error getting relative path: %v\n", err)
		return false
	}

	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/") {
			if strings.HasPrefix(relativePath, strings.TrimSuffix(pattern, "/")) {
				return true
			}
		} else if matched, _ := filepath.Match(pattern, relativePath); matched {
			return true
		} else if strings.Contains(pattern, "*") {
			matched, _ := filepath.Match(pattern, relativePath)
			if matched {
				return true
			}
		} else if strings.HasPrefix(relativePath, pattern) {
			return true
		}
	}

	return false
}

func readFileAndScan(path string, rules []*regexp.Regexp, matches chan<- string, wg *sync.WaitGroup, bar *progressbar.ProgressBar) {
	defer wg.Done()

	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", path, err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error reading file %s: %v\n", path, err)
			}
			break
		}
		processLine(path, line, rules, matches, bar)
	}

	bar.Add(1) // Increment after processing the file
}

func processLine(path string, line []byte, rules []*regexp.Regexp, matches chan<- string, bar *progressbar.ProgressBar) {
	for _, rule := range rules {
		if rule.Match(line) {
			matches <- fmt.Sprintf("Potential secret found in file %s: %s", path, line)
		}
	}
	bar.Add(1)
}

func scanDirectory(dirPath string, cfg config.Config) {
	defer trackTime(time.Now(), "scanDirectory")

	var totalLines int
	patterns, err := parseGitignore(dirPath)
	if err != nil {
		fmt.Printf("Error reading .gitignore: %v\n", err)
		os.Exit(1)
	}

	// Count total lines across all files
	err = filepath.WalkDir(dirPath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.HasSuffix(info.Name(), ".env") || shouldIgnore(path, dirPath, patterns) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewReader(file)
		for {
			_, err := scanner.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Error reading file %s: %v\n", path, err)
				}
				break
			}

			totalLines++
		}

		return err
	})

	if err != nil {
		fmt.Printf("Error counting lines: %v\n", err)
		os.Exit(1)
	}

	// Initialize progress bar with total lines
	bar := progressbar.Default(int64(totalLines), "Scanning")

	var rules []*regexp.Regexp
	for _, rule := range cfg.Rules {
		re := regexp.MustCompile(rule.Regex)
		rules = append(rules, re)
	}

	// Channel to collect matches
	matches := make(chan string)

	// WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Start goroutines to scan files
	err = filepath.WalkDir(dirPath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.HasSuffix(info.Name(), ".env") || shouldIgnore(path, dirPath, patterns) {
			return nil
		}

		wg.Add(1)
		go readFileAndScan(path, rules, matches, &wg, bar)

		return nil
	})

	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	// Close the matches channel once all goroutines are done
	go func() {
		wg.Wait()
		close(matches)
	}()

	// Collect and print matches
	var allMatches []string
	for match := range matches {
		allMatches = append(allMatches, match)
	}

	fmt.Println() // Ensure newline after progress bar

	if len(allMatches) > 0 {
		fmt.Println("Potential secrets found:")
		for _, match := range allMatches {
			fmt.Println(match)
		}
		report.GenerateReport(allMatches, "json")

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
