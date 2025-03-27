package cmd

import (
	"bufio"
	"bytes" // Added for bytes.IndexByte
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"envscan/packages/config"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// Package-level variables
var configFile string
var failFast bool
var showProgress bool
var followSymlinks bool

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check staged files or a commit range for secrets",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dirPath := args[0]
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			log.Printf("Error loading config %s: %v. Try specifying a valid TOML file with --config.", configFile, err)
			os.Exit(1)
		}
		scanDirectory(dirPath, cfg)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(changedFilesCmd)

	checkCmd.Flags().StringVarP(&configFile, "config", "c", "secrets.toml", "Path to the configuration file")
	changedFilesCmd.Flags().StringVarP(&configFile, "config", "c", "secrets.toml", "Path to the configuration file")

	checkCmd.Flags().BoolVar(&failFast, "fail-fast", false, "Exit immediately on first error")
	changedFilesCmd.Flags().BoolVar(&failFast, "fail-fast", false, "Exit immediately on first error")

	checkCmd.Flags().BoolVar(&showProgress, "no-progress", true, "Show progress bar")
	changedFilesCmd.Flags().BoolVar(&showProgress, "no-progress", true, "Show progress bar")

	checkCmd.Flags().BoolVar(&followSymlinks, "follow-symlinks", true, "Follow symbolic links")
	changedFilesCmd.Flags().BoolVar(&followSymlinks, "follow-symlinks", true, "Follow symbolic links")
}

func parseGitignore(dirPath string) (*ignore.GitIgnore, error) {
	gitignorePath := filepath.Join(dirPath, ".gitignore")
	return ignore.CompileIgnoreFile(gitignorePath)
}

func shouldIgnore(path string, dirPath string, gi *ignore.GitIgnore) bool {
	if gi == nil {
		return false
	}
	relPath, _ := filepath.Rel(dirPath, path)
	return gi.MatchesPath(relPath)
}

func readFileAndScan(path string, rules []*regexp.Regexp, matches chan<- string, wg *sync.WaitGroup, bar *progressbar.ProgressBar) {
	defer wg.Done()

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file %s: %v", path, err)
		return
	}
	defer file.Close()

	// Check if the file is binary
	if isBinaryFile(file) {
		return
	}

	// Reset file pointer to the beginning after checking
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		log.Printf("Error seeking file %s: %v", path, err)
		return
	}

	// Proceed with reading the file line by line
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading file %s: %v", path, err)
			}
			break
		}
		processLine(path, line, rules, matches, bar)
	}

	bar.Add(1)
}

func isBinaryFile(file *os.File) bool {
	const checkSize = 8000
	buf := make([]byte, checkSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		log.Printf("Error reading file %s: %v", file.Name(), err)
		return false
	}
	return bytes.IndexByte(buf[:n], 0) != -1
}

func processLine(path string, line []byte, rules []*regexp.Regexp, matches chan<- string, bar *progressbar.ProgressBar) {
	exclusions := []*regexp.Regexp{
		regexp.MustCompile(`process\.env\.`),
		regexp.MustCompile(`os\.environ\['`),
		regexp.MustCompile(`ENV\['`),
		regexp.MustCompile(`System\.getenv\(`),
		regexp.MustCompile(`getenv\(`),
		regexp.MustCompile(`\$ENV\{`),
		regexp.MustCompile(`System\.Environment`),
		regexp.MustCompile(`dotenv\.`),
		regexp.MustCompile(`config\.`),
	}

	for _, exclusion := range exclusions {
		if exclusion.Match(line) {
			return
		}
	}

	for _, rule := range rules {
		if rule.Match(line) {
			matches <- fmt.Sprintf("Potential secret found in file %s: %s", path, line)
		}
	}
	bar.Add(1)
}

func scanDirectory(dirPath string, cfg config.Config) {
	var totalLines int
	gi, err := parseGitignore(dirPath)
	if err != nil {
		log.Printf("Error reading .gitignore: %v\n", err)
		os.Exit(1)
	}

	err = filepath.WalkDir(dirPath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !followSymlinks && info.Type()&os.ModeSymlink != 0 {
			return nil
		}

		if info.IsDir() || shouldIgnore(path, dirPath, gi) {
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
					log.Printf("Error reading file %s: %v\n", path, err)
				}
				break
			}
			totalLines++
		}

		return err
	})

	if err != nil {
		log.Printf("Error counting lines: %v\n", err)
		os.Exit(1)
	}

	var bar *progressbar.ProgressBar
	if showProgress {
		bar = progressbar.Default(int64(totalLines), "Scanning")
	}

	var rules []*regexp.Regexp
	for _, rule := range cfg.Rules {
		re := regexp.MustCompile(rule.Regex)
		rules = append(rules, re)
	}

	matches := make(chan string)
	var wg sync.WaitGroup

	err = filepath.WalkDir(dirPath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || shouldIgnore(path, dirPath, gi) {
			return nil
		}

		wg.Add(1)
		go readFileAndScan(path, rules, matches, &wg, bar)

		return nil
	})

	if err != nil {
		log.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	go func() {
		wg.Wait()
		close(matches)
	}()

	var allMatches []string
	for match := range matches {
		allMatches = append(allMatches, match)
	}

	log.Println()

	if len(allMatches) > 0 {
		log.Println("Potential secrets found:")
		for _, match := range allMatches {
			log.Println(match)
		}
		os.Exit(1)
	} else {
		log.Println("No secrets found")
	}
}

// Command for checking only changed files
var changedFilesCmd = &cobra.Command{
	Use:   "changed",
	Short: "Check only changed files for secrets",
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			log.Printf("Error loading config %s: %v. Try specifying a valid TOML file with --config.", configFile, err)
			os.Exit(1)
		}

		// Get list of changed files
		changedFiles, err := getCommittedFiles()
		if err != nil {
			log.Printf("Error getting changed files: %v", err)
			os.Exit(1)
		}

		// Scan only changed files
		scanCommittedFiles(changedFiles, cfg)
	},
}

// Helper function to get the root directory of the git repository
func getGitRootDirectory() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git repository root: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Helper function to get list of changed files
func getCommittedFiles() ([]string, error) {
	// Get root directory to construct full paths
	rootDir, err := getGitRootDirectory()
	if err != nil {
		return nil, err
	}

	// Use git ls-files to get changed files
	cmd := exec.Command("git", "ls-files")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %v", err)
	}

	// Convert relative paths to full paths
	var changedFiles []string
	for _, file := range strings.Split(string(output), "\n") {
		if file != "" {
			changedFiles = append(changedFiles, filepath.Join(rootDir, file))
		}
	}

	return changedFiles, nil
}

// Specialized scan for changed files
func scanCommittedFiles(files []string, cfg config.Config) {
	var totalLines int
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			log.Printf("Error opening file %s: %v", file, err)
			continue
		}
		defer f.Close()

		scanner := bufio.NewReader(f)
		for {
			_, err := scanner.ReadBytes('\n')
			if err != nil {
				break
			}
			totalLines++
		}
	}

	var bar *progressbar.ProgressBar
	if showProgress {
		bar = progressbar.Default(int64(totalLines), "Scanning Committed Files")
	}

	var rules []*regexp.Regexp
	for _, rule := range cfg.Rules {
		re := regexp.MustCompile(rule.Regex)
		rules = append(rules, re)
	}

	matches := make(chan string)
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)
		go readFileAndScan(file, rules, matches, &wg, bar)
	}

	go func() {
		wg.Wait()
		close(matches)
	}()

	var allMatches []string
	for match := range matches {
		allMatches = append(allMatches, match)
	}

	log.Println()

	if len(allMatches) > 0 {
		log.Println("Potential secrets found in changed files:")
		for _, match := range allMatches {
			log.Println(match)
		}
		os.Exit(1)
	} else {
		log.Println("No secrets found in changed files")
	}
}
