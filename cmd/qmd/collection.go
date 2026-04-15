package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aliumujib/qmd-go/pkg/config"
	"github.com/aliumujib/qmd-go/pkg/indexer"
	"github.com/spf13/cobra"
)

var collectionCmd = &cobra.Command{
	Use:   "collection",
	Short: "Manage collections",
}

var collectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collections",
	Run: func(cmd *cobra.Command, args []string) {
		initRoot()
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		if len(cfg.Collections) == 0 {
			fmt.Println("No collections found.")
			return
		}

		fmt.Println("Collections:")
		for name, col := range cfg.Collections {
			fmt.Printf("- %s (%s) [%s]\n", name, col.Path, col.Pattern)
		}
	},
}

var collectionAddCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Add a collection",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initRoot()
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Printf("Invalid path: %v\n", err)
			os.Exit(1)
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = filepath.Base(absPath)
		}
		pattern, _ := cmd.Flags().GetString("mask")
		if pattern == "" {
			pattern = "**/*.md"
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		if _, exists := cfg.Collections[name]; exists {
			fmt.Printf("Collection '%s' already exists.\n", name)
			os.Exit(1)
		}

		cfg.Collections[name] = config.Collection{
			Path:    absPath,
			Pattern: pattern,
		}

		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}

		s, err := openStore()
		if err != nil {
			fmt.Printf("Error opening store: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()
		fmt.Printf("Indexing collection '%s'...\n", name)
		if err := indexer.IndexFiles(s, name, absPath, pattern); err != nil {
			fmt.Printf("Error indexing: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Collection '%s' added.\n", name)
	},
}

var collectionRemoveCmd = &cobra.Command{
	Use:     "remove [name]",
	Aliases: []string{"rm"},
	Short:   "Remove a collection",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initRoot()
		name := args[0]
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		if _, exists := cfg.Collections[name]; !exists {
			fmt.Printf("Collection '%s' not found.\n", name)
			os.Exit(1)
		}

		delete(cfg.Collections, name)
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		s, err := openStore()
		if err == nil {
			_, _ = s.DB.Exec(`DELETE FROM documents WHERE collection = ?`, name)
			s.Close()
		}
		fmt.Printf("Collection '%s' removed.\n", name)
	},
}

var collectionRenameCmd = &cobra.Command{
	Use:     "rename <old> <new>",
	Aliases: []string{"mv"},
	Short:   "Rename a collection",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		initRoot()
		oldName, newName := args[0], args[1]
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		if _, exists := cfg.Collections[oldName]; !exists {
			fmt.Printf("Collection '%s' not found.\n", oldName)
			os.Exit(1)
		}
		if _, exists := cfg.Collections[newName]; exists {
			fmt.Printf("Collection '%s' already exists.\n", newName)
			os.Exit(1)
		}
		cfg.Collections[newName] = cfg.Collections[oldName]
		delete(cfg.Collections, oldName)
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		s, err := openStore()
		if err != nil {
			fmt.Printf("Error opening store: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()
		_, err = s.DB.Exec(`UPDATE documents SET collection = ? WHERE collection = ?`, newName, oldName)
		if err != nil {
			fmt.Printf("Error updating documents: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Renamed '%s' to '%s' (qmd://%s/)\n", oldName, newName, newName)
	},
}

func init() {
	collectionAddCmd.Flags().String("name", "", "Collection name")
	collectionAddCmd.Flags().String("mask", "**/*.md", "File pattern mask")

	collectionCmd.AddCommand(collectionListCmd)
	collectionCmd.AddCommand(collectionAddCmd)
	collectionCmd.AddCommand(collectionRemoveCmd)
	collectionCmd.AddCommand(collectionRenameCmd)
	rootCmd.AddCommand(collectionCmd)
}
