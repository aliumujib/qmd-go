package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aliumujib/qmd-go/pkg/llm"
	"github.com/aliumujib/qmd-go/pkg/store"
	"github.com/spf13/cobra"
)

func formatDocForEmbedding(text, title string) string {
	if title == "" {
		title = "none"
	}
	return "title: " + title + " | text: " + text
}

func extractTitle(body string) string {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
		return line
	}
	return ""
}

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Generate vector embeddings",
	Long:  "Generate vector embeddings for indexed documents using Ollama or OpenAI-compatible API.",
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")
		model := os.Getenv("QMD_EMBED_MODEL")
		if model == "" {
			model = llm.DefaultEmbedModel()
		}

		initRoot()
		s, err := openStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening store: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()

		if err := s.EnsureEmbeddingBlobTable(); err != nil {
			fmt.Fprintf(os.Stderr, "Error ensuring embedding table: %v\n", err)
			os.Exit(1)
		}

		if force {
			fmt.Fprintln(os.Stderr, "Force re-embedding: clearing all vectors...")
			if err := s.ClearAllEmbeddings(); err != nil {
				fmt.Fprintf(os.Stderr, "Error clearing embeddings: %v\n", err)
				os.Exit(1)
			}
		}

		hashes, err := s.GetHashesForEmbedding()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting hashes: %v\n", err)
			os.Exit(1)
		}
		if len(hashes) == 0 {
			fmt.Println("All content hashes already have embeddings.")
			return
		}

		backend := os.Getenv("QMD_EMBED_BACKEND")
		if backend == "" {
			if llm.GGUFEnabled() {
				backend = "gguf (default)"
			} else {
				backend = "api (default)"
			}
		}
		client, err := llm.NewEmbedClient(model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating embed client: %v\n", err)
			os.Exit(1)
		}

		var totalChunks int
		for _, h := range hashes {
			chunks := store.ChunkDocument(h.Body, store.ChunkSizeChars, store.ChunkOverlapChars)
			totalChunks += len(chunks)
		}
		fmt.Printf("Embedding %d documents (%d chunks), model: %s, backend: %s\n\n", len(hashes), totalChunks, model, backend)

		embedded := 0
		errors := 0
		now := time.Now()

		for i, h := range hashes {
			docTitle := extractTitle(h.Body)
			if docTitle == "" {
				parts := strings.Split(h.Path, "/")
				if len(parts) > 0 {
					docTitle = parts[len(parts)-1]
				}
			}
			chunks := store.ChunkDocument(h.Body, store.ChunkSizeChars, store.ChunkOverlapChars)
			for seq, ch := range chunks {
				formatted := formatDocForEmbedding(ch.Text, docTitle)
				result, err := client.Embed(formatted)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error embedding %s chunk %d: %v\n", h.Path, seq, err)
					errors++
					continue
				}
				if err := s.InsertEmbedding(h.Hash, seq, ch.Pos, result.Embedding, model, now); err != nil {
					fmt.Fprintf(os.Stderr, "Error inserting embedding: %v\n", err)
					errors++
					continue
				}
				embedded++
			}
			fmt.Fprintf(os.Stderr, "\rEmbedded %d/%d documents (%d chunks)...", i+1, len(hashes), embedded)
		}
		fmt.Fprintln(os.Stderr)
		elapsed := time.Since(now).Seconds()
		fmt.Printf("Done. Embedded %d chunks from %d documents in %.1fs", embedded, len(hashes), elapsed)
		if errors > 0 {
			fmt.Printf(" (%d errors)", errors)
		}
		fmt.Println()
	},
}

func init() {
	embedCmd.Flags().BoolP("force", "f", false, "Force re-embedding (clear all vectors first)")
	rootCmd.AddCommand(embedCmd)
}
