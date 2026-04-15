package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliumujib/qmd-go/pkg/config"
	"github.com/aliumujib/qmd-go/pkg/store"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage path context descriptions",
}

var contextAddCmd = &cobra.Command{
	Use:   "add [path] \"text\"",
	Short: "Add context for path (defaults to current dir)",
	Long:  "Add context for a path. Use '/' for global context. Use qmd://collection/path for virtual paths.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initRoot()
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		var pathArg, contextText string
		if len(args) >= 2 && (strings.HasPrefix(args[0], "qmd://") || args[0] == "/" || args[0] == "." || strings.Contains(args[0], "/")) {
			pathArg = args[0]
			contextText = strings.Join(args[1:], " ")
		} else {
			contextText = strings.Join(args, " ")
		}

		if pathArg == "/" {
			config.SetGlobalContext(cfg, contextText)
			if err := config.SaveConfig(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Set global context")
			return
		}

		if pathArg == "" {
			pathArg = "."
		}
		pathArg = strings.TrimSuffix(pathArg, "/")

		// Virtual path: qmd://collection/path
		if strings.HasPrefix(pathArg, "qmd://") {
			rest := strings.TrimPrefix(pathArg, "qmd://")
			idx := strings.Index(rest, "/")
			collectionName := rest
			pathPrefix := "/"
			if idx >= 0 {
				collectionName = rest[:idx]
				pathPrefix = "/" + rest[idx+1:]
			}
			if !config.AddContext(cfg, collectionName, pathPrefix, contextText) {
				fmt.Fprintf(os.Stderr, "Collection not found: %s\n", collectionName)
				os.Exit(1)
			}
			if err := config.SaveConfig(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Added context for qmd://%s%s\n", collectionName, pathPrefix)
			return
		}

		// Resolve filesystem path to collection + path
		absPath, err := filepath.Abs(pathArg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid path: %v\n", err)
			os.Exit(1)
		}
		initRoot()
		s, err := openStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening store: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()

		collectionName, relativePath := detectCollectionFromPath(s, absPath, cfg)
		if collectionName == "" {
			fmt.Fprintf(os.Stderr, "Path is not in any indexed collection: %s\n", absPath)
			os.Exit(1)
		}
		pathPrefix := "/"
		if relativePath != "" {
			pathPrefix = "/" + relativePath
		}
		if !config.AddContext(cfg, collectionName, pathPrefix, contextText) {
			fmt.Fprintf(os.Stderr, "Collection not found: %s\n", collectionName)
			os.Exit(1)
		}
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added context for qmd://%s%s\n", collectionName, pathPrefix)
	},
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contexts",
	Run: func(cmd *cobra.Command, args []string) {
		initRoot()
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		entries := config.ListAllContexts(cfg)
		if len(entries) == 0 {
			fmt.Println("No contexts configured. Use 'qmd context add' to add one.")
			return
		}
		fmt.Println("\nConfigured Contexts")
		fmt.Println()
		lastCol := ""
		for _, e := range entries {
			if e.Collection != lastCol {
				fmt.Println(e.Collection)
				lastCol = e.Collection
			}
			p := e.Path
			if p == "/" {
				p = "  / (root)"
			} else {
				p = "  " + p
			}
			fmt.Println(p)
			fmt.Println("    " + e.Context)
		}
	},
}

var contextCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for collections or paths missing context",
	Run: func(cmd *cobra.Command, args []string) {
		initRoot()
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		var without []string
		for name, col := range cfg.Collections {
			if col.Context == nil || len(col.Context) == 0 {
				without = append(without, name)
			}
		}
		if len(without) == 0 && len(cfg.Collections) > 0 {
			fmt.Println("All collections have context configured.")
		}
		if len(without) > 0 {
			fmt.Println("Collections without any context:")
			for _, name := range without {
				fmt.Printf("  %s\n", name)
				fmt.Printf("    Suggestion: qmd context add qmd://%s/ \"Description\"\n", name)
			}
		}
	},
}

var contextRmCmd = &cobra.Command{
	Use:   "rm <path>",
	Short: "Remove context",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initRoot()
		pathArg := args[0]
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if pathArg == "/" {
			config.SetGlobalContext(cfg, "")
			if err := config.SaveConfig(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Removed global context")
			return
		}

		if strings.HasPrefix(pathArg, "qmd://") {
			rest := strings.TrimPrefix(pathArg, "qmd://")
			idx := strings.Index(rest, "/")
			collectionName := rest
			pathPrefix := "/"
			if idx >= 0 {
				collectionName = rest[:idx]
				pathPrefix = "/" + rest[idx+1:]
			}
			if !config.RemoveContext(cfg, collectionName, pathPrefix) {
				fmt.Fprintf(os.Stderr, "No context found for: %s\n", pathArg)
				os.Exit(1)
			}
			if err := config.SaveConfig(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Removed context for %s\n", pathArg)
			return
		}

		absPath, _ := filepath.Abs(pathArg)
		initRoot()
		s, err := openStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening store: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()
		collectionName, relativePath := detectCollectionFromPath(s, absPath, cfg)
		if collectionName == "" {
			fmt.Fprintf(os.Stderr, "Path is not in any indexed collection: %s\n", pathArg)
			os.Exit(1)
		}
		pathPrefix := "/"
		if relativePath != "" {
			pathPrefix = "/" + relativePath
		}
		if !config.RemoveContext(cfg, collectionName, pathPrefix) {
			fmt.Fprintf(os.Stderr, "No context found for qmd://%s%s\n", collectionName, pathPrefix)
			os.Exit(1)
		}
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed context for qmd://%s%s\n", collectionName, pathPrefix)
	},
}

func detectCollectionFromPath(s *store.Store, absPath string, cfg *config.Config) (collectionName, relativePath string) {
	var bestName, bestRel string
	bestLen := 0
	for name, col := range cfg.Collections {
		colPath, err := filepath.Abs(col.Path)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(colPath, absPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		rel = filepath.ToSlash(rel)
		if len(col.Path) > bestLen {
			bestLen = len(col.Path)
			bestName = name
			bestRel = rel
		}
	}
	return bestName, bestRel
}

func init() {
	contextCmd.AddCommand(contextAddCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextCheckCmd)
	contextCmd.AddCommand(contextRmCmd)
	rootCmd.AddCommand(contextCmd)
}
