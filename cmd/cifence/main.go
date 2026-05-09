package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oaslananka/cifence/internal/analyzer"
	"github.com/oaslananka/cifence/internal/config"
	"github.com/oaslananka/cifence/internal/githubactions"
	"github.com/oaslananka/cifence/internal/report"
	"github.com/oaslananka/cifence/internal/rules"
	"github.com/oaslananka/cifence/internal/sarif"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, analyzer.Version)
		return 0
	case "rules":
		for _, definition := range rules.Definitions {
			if definition.ID == rules.ParseRuleID {
				continue
			}
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", definition.ID, definition.Severity, definition.Title)
		}
		return 0
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func runScan(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("scan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	pathFlag := flags.String("path", ".", "repository or workflow path to scan")
	formatFlag := flags.String("format", "markdown", "output format: markdown, json, sarif")
	modeFlag := flags.String("mode", "warn", "execution mode: warn, enforce")
	failOnFlag := flags.String("fail-on", "", "minimum severity that fails enforce mode: low, medium, high, critical")
	configPath := flags.String("config", "", "path to cifence.yml policy config")
	baselinePath := flags.String("baseline", "", "path to CIFence baseline JSON")
	updateBaseline := flags.Bool("update-baseline", false, "write current findings to the baseline file")
	jsonPath := flags.String("json", "", "write JSON report to path")
	sarifPath := flags.String("sarif", "", "write SARIF report to path")
	markdownPath := flags.String("markdown", "", "write Markdown report to path")
	normalizedArgs, ok := normalizeScanArgs(args, stderr)
	if !ok {
		return 2
	}
	if err := flags.Parse(normalizedArgs); err != nil {
		return 2
	}

	path := *pathFlag
	if flags.NArg() > 1 {
		fmt.Fprintln(stderr, "scan accepts at most one positional path")
		return 2
	}
	if flags.NArg() == 1 {
		path = flags.Arg(0)
	}
	if !validFormat(*formatFlag) {
		fmt.Fprintf(stderr, "invalid format %q\n", *formatFlag)
		return 2
	}
	if *modeFlag != "warn" && *modeFlag != "enforce" {
		fmt.Fprintf(stderr, "invalid mode %q\n", *modeFlag)
		return 2
	}
	if *updateBaseline && *baselinePath == "" {
		fmt.Fprintln(stderr, "--update-baseline requires --baseline")
		return 2
	}

	cfg, _, err := config.Load(path, *configPath)
	if err != nil {
		fmt.Fprintf(stderr, "config failed: %v\n", err)
		return 1
	}
	failOn := cfg.FailOn()
	if *failOnFlag != "" {
		if !githubactions.ValidSeverity(*failOnFlag) {
			fmt.Fprintf(stderr, "invalid fail-on %q\n", *failOnFlag)
			return 2
		}
		failOn = githubactions.Severity(*failOnFlag)
	}

	result, err := analyzer.ScanWithOptions(path, analyzer.ScanOptions{
		Config:         cfg,
		HasConfig:      true,
		BaselinePath:   *baselinePath,
		UpdateBaseline: *updateBaseline,
	})
	if err != nil {
		fmt.Fprintf(stderr, "scan failed: %v\n", err)
		return 1
	}

	jsonBytes, err := report.JSON(result)
	if err != nil {
		fmt.Fprintf(stderr, "json report failed: %v\n", err)
		return 1
	}
	sarifBytes, err := sarif.JSON(result)
	if err != nil {
		fmt.Fprintf(stderr, "sarif report failed: %v\n", err)
		return 1
	}
	markdown := report.Markdown(result)

	if err := writeOptional(*jsonPath, jsonBytes); err != nil {
		fmt.Fprintf(stderr, "write json report failed: %v\n", err)
		return 1
	}
	if err := writeOptional(*sarifPath, sarifBytes); err != nil {
		fmt.Fprintf(stderr, "write sarif report failed: %v\n", err)
		return 1
	}
	if err := writeOptional(*markdownPath, []byte(markdown)); err != nil {
		fmt.Fprintf(stderr, "write markdown report failed: %v\n", err)
		return 1
	}

	switch *formatFlag {
	case "json":
		fmt.Fprintln(stdout, string(jsonBytes))
	case "sarif":
		fmt.Fprintln(stdout, string(sarifBytes))
	case "markdown":
		fmt.Fprint(stdout, markdown)
	}

	if *modeFlag == "enforce" && analyzer.EnforceFailsAt(result, failOn) {
		return 1
	}
	return 0
}

func validFormat(value string) bool {
	return value == "markdown" || value == "json" || value == "sarif"
}

func normalizeScanArgs(args []string, stderr io.Writer) ([]string, bool) {
	knownValueFlags := map[string]struct{}{
		"--path":     {},
		"--format":   {},
		"--mode":     {},
		"--fail-on":  {},
		"--config":   {},
		"--baseline": {},
		"--json":     {},
		"--sarif":    {},
		"--markdown": {},
	}

	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, 1)
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if _, ok := knownValueFlags[arg]; ok {
			if index+1 >= len(args) {
				fmt.Fprintf(stderr, "%s requires a value\n", arg)
				return nil, false
			}
			flags = append(flags, arg, args[index+1])
			index++
			continue
		}
		if strings.HasPrefix(arg, "--") {
			flags = append(flags, arg)
			continue
		}
		positionals = append(positionals, arg)
	}
	return append(flags, positionals...), true
}

func writeOptional(path string, content []byte) error {
	if path == "" {
		return nil
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return os.WriteFile(path, append(content, '\n'), 0o600)
}

func printHelp(writer io.Writer) {
	fmt.Fprintln(writer, "CIFence")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  cifence scan [path] [--format markdown|json|sarif] [--mode warn|enforce] [--fail-on low|medium|high|critical]")
	fmt.Fprintln(writer, "  cifence scan --path . --json cifence.json --sarif cifence.sarif --markdown cifence.md")
	fmt.Fprintln(writer, "  cifence scan --baseline cifence.baseline.json --update-baseline")
	fmt.Fprintln(writer, "  cifence version")
	fmt.Fprintln(writer, "  cifence rules")
	fmt.Fprintln(writer, "  cifence help")
}
