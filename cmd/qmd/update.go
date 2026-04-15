package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aliumujib/qmd-go/pkg/config"
	"github.com/aliumujib/qmd-go/pkg/indexer"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Re-index all collections",
	Run: func(cmd *cobra.Command, args []string) {
		initRoot()
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}
		if len(cfg.Collections) == 0 {
			fmt.Println("No collections. Run 'qmd collection add .' to index files.")
			return
		}

		pull, _ := cmd.Flags().GetBool("pull")
		_, _ = cmd.Flags().GetBool("full") // accept for compatibility; full re-index is the default
		s, err := openStore()
		if err != nil {
			fmt.Printf("Error opening store: %v\n", err)
			return
		}
		defer s.Close()

		for name, col := range cfg.Collections {
			if pull {
				// Run git pull in collection root if it's a git repo
				if _, err := os.Stat(filepath.Join(col.Path, ".git")); err == nil {
					c := exec.Command("git", "pull")
					c.Dir = col.Path
					c.Stdout = os.Stdout
					c.Stderr = os.Stderr
					if runErr := c.Run(); runErr != nil {
						fmt.Printf("Warning: git pull in %s failed: %v\n", col.Path, runErr)
					}
				}
			}
			fmt.Printf("Updating collection '%s'...\n", name)
			if err := indexer.IndexFiles(s, name, col.Path, col.Pattern); err != nil {
				fmt.Printf("Error indexing collection '%s': %v\n", name, err)
			}
		}
		fmt.Println("Done.")
	},
}

func init() {
	updateCmd.Flags().Bool("pull", false, "Run git pull in each collection root before re-indexing")
	updateCmd.Flags().Bool("full", false, "Full re-index (default behavior; accepted for compatibility)")
	rootCmd.AddCommand(updateCmd)
}
