package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Xavier-GAO-42/clewkeep/internal/app"
	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

const version = "0.1.0-rc.1"

type scanFlags struct {
	full bool
	json bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		usage()
		return nil
	}
	application, err := app.New()
	if err != nil {
		return err
	}
	ctx := context.Background()
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println("ctx", version)
	case "scan":
		flags, err := parseScanFlags(args[1:])
		if err != nil {
			return err
		}
		var catalog *core.Catalog
		if flags.full {
			catalog, err = application.ScanFull(ctx)
		} else {
			catalog, err = application.Scan(ctx)
		}
		if err != nil {
			return err
		}
		if flags.json {
			return printJSON(catalog)
		}
		providers, projects := summarize(catalog.Threads)
		fmt.Printf("scanned: %d thread(s), %d provider(s), %d project(s)\n", len(catalog.Threads), providers, projects)
		fmt.Println("catalog:", application.StoreHome)
	case "status":
		status, err := application.Status()
		if err != nil {
			return err
		}
		if hasFlag(args[1:], "--json") {
			return printJSON(status)
		}
		fmt.Println("catalog:", status.CatalogPath)
		fmt.Println("generated:", status.GeneratedAt)
		fmt.Printf("threads: %d  providers: %d  projects: %d  warnings: %d\n", status.Threads, len(status.Providers), status.Projects, status.Warnings)
	case "list", "ls":
		threads, err := application.List(option(args[1:], "--provider"), option(args[1:], "--project"))
		if err != nil {
			return err
		}
		if hasFlag(args[1:], "--json") {
			return printJSON(threads)
		}
		for _, thread := range threads {
			fmt.Printf("%s  %s  %s  lines=%d\n", thread.ID, thread.Environment, thread.ProjectRoot, thread.LineCount)
		}
	case "show":
		positionals := positional(args[1:], map[string]bool{"--json": false})
		if len(positionals) != 1 {
			return fmt.Errorf("usage: ctx show <id-or-name> [--json]")
		}
		thread, name, err := application.Show(positionals[0])
		if err != nil {
			return err
		}
		if hasFlag(args[1:], "--json") {
			return printJSON(struct {
				Name   string `json:"name,omitempty"`
				Thread any    `json:"thread"`
			}{name, thread})
		}
		if name != "" {
			fmt.Println("name:", name)
		}
		fmt.Println("thread:", thread.ID)
		fmt.Println("provider:", thread.Provider)
		fmt.Println("environment:", thread.Environment)
		fmt.Println("project:", thread.ProjectRoot)
		fmt.Println("native:", thread.NativePath)
		fmt.Println("format:", thread.NativeFormat)
		fmt.Println("lines:", thread.LineCount)
	case "name":
		if len(args) < 3 {
			return fmt.Errorf("usage: ctx name <id-or-name> <name>")
		}
		name := strings.Join(args[2:], " ")
		thread, err := application.Name(args[1], name)
		if err != nil {
			return err
		}
		fmt.Printf("named: %s -> %s\n", name, thread.ID)
	case "search":
		positionals := positional(args[1:], map[string]bool{"--json": false, "--limit": true})
		if len(positionals) == 0 {
			return fmt.Errorf("usage: ctx search <query> [--limit <n>] [--json]")
		}
		limit := 20
		if value := option(args[1:], "--limit"); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed <= 0 || parsed > 500 {
				return fmt.Errorf("--limit must be between 1 and 500")
			}
			limit = parsed
		}
		hits, err := application.Search(strings.Join(positionals, " "), limit)
		if err != nil {
			return err
		}
		if hasFlag(args[1:], "--json") {
			return printJSON(hits)
		}
		for _, hit := range hits {
			label := hit.ThreadID
			if hit.Name != "" {
				label = hit.Name + " (" + hit.ThreadID + ")"
			}
			fmt.Printf("%s  %s  %s:%d\n  %s\n", label, hit.Provider, hit.NativePath, hit.Line, hit.Snippet)
		}
		fmt.Printf("%d result(s)\n", len(hits))
	case "snapshot":
		path, snapshot, err := application.Snapshot(ctx, option(args[1:], "--name"))
		if err != nil {
			return err
		}
		if hasFlag(args[1:], "--json") {
			return printJSON(snapshot)
		}
		fmt.Println("snapshot:", path)
		fmt.Println("threads:", len(snapshot.Threads))
	case "diff":
		if len(args) < 2 || args[1] != "--since" {
			return fmt.Errorf("usage: ctx diff --since [latest|snapshot] [--json]")
		}
		selector := "latest"
		for _, value := range args[2:] {
			if !strings.HasPrefix(value, "--") {
				selector = value
				break
			}
		}
		diff, err := application.DiffSince(ctx, selector)
		if err != nil {
			return err
		}
		if hasFlag(args[1:], "--json") {
			return printJSON(diff)
		}
		fmt.Printf("change: +%d  ~%d  -%d  =%d\n", len(diff.Added), len(diff.Updated), len(diff.Removed), diff.Unchanged)
		for _, thread := range diff.Added {
			fmt.Printf("+ %s  %s  %s\n", thread.ID, thread.Environment, thread.ProjectRoot)
		}
		for _, change := range diff.Updated {
			fmt.Printf("~ %s  %s  lines=%d->%d\n", change.After.ID, change.After.Environment, change.Before.LineCount, change.After.LineCount)
		}
		for _, thread := range diff.Removed {
			fmt.Printf("- %s  %s  %s\n", thread.ID, thread.Environment, thread.ProjectRoot)
		}
	case "doctor":
		checks := application.Doctor()
		if hasFlag(args[1:], "--json") {
			return printJSON(checks)
		}
		for _, check := range checks {
			fmt.Printf("%-24s %-10s %s\n", check.Name, check.Status, check.Detail)
		}
	default:
		return fmt.Errorf("unknown command %q; run ctx --help", args[0])
	}
	return nil
}

func usage() {
	fmt.Print(`ctx 0.1 — local, read-first agent context catalog

Usage:
  ctx scan [--full] [--json]
  ctx status [--json]
  ctx list [--provider <name>] [--project <text>] [--json]
  ctx search <query> [--limit <n>] [--json]
  ctx show <id-or-name> [--json]
  ctx name <id-or-name> <name>
  ctx snapshot [--name <name>] [--json]
  ctx diff --since [latest|snapshot] [--json]
  ctx doctor [--json]
  ctx version
`)
}

func parseScanFlags(args []string) (scanFlags, error) {
	var flags scanFlags
	for _, arg := range args {
		switch arg {
		case "--full":
			flags.full = true
		case "--json":
			flags.json = true
		default:
			return scanFlags{}, fmt.Errorf("unknown scan argument %q; usage: ctx scan [--full] [--json]", arg)
		}
	}
	return flags, nil
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func hasFlag(args []string, flag string) bool {
	for _, value := range args {
		if value == flag {
			return true
		}
	}
	return false
}

func option(args []string, name string) string {
	for index, value := range args {
		if value == name && index+1 < len(args) {
			return args[index+1]
		}
		if strings.HasPrefix(value, name+"=") {
			return strings.TrimPrefix(value, name+"=")
		}
	}
	return ""
}

func positional(args []string, flags map[string]bool) []string {
	result := make([]string, 0)
	for index := 0; index < len(args); index++ {
		value := args[index]
		if consumes, exists := flags[value]; exists {
			if consumes {
				index++
			}
			continue
		}
		isInlineFlag := false
		for flag := range flags {
			if strings.HasPrefix(value, flag+"=") {
				isInlineFlag = true
				break
			}
		}
		if !isInlineFlag {
			result = append(result, value)
		}
	}
	return result
}

func summarize(threads []core.Thread) (int, int) {
	providers := map[string]bool{}
	projects := map[string]bool{}
	for _, thread := range threads {
		providers[thread.Provider] = true
		projects[thread.Provider+"\x00"+thread.ProjectRoot] = true
	}
	return len(providers), len(projects)
}
