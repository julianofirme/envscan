package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

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
	Rules []Rule `toml:"rules"`
}

var configFile string

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
		os.Exit(1)
	} else {
		fmt.Println("No secrets found")
	}
}
