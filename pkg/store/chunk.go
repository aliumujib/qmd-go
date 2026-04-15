package store

// ChunkSizeChars and ChunkOverlapChars match TS defaults (~800 tokens * 4 chars, 15% overlap).
const (
	ChunkSizeChars    = 3200
	ChunkOverlapChars = 480
)

// Chunk is one piece of document text with its start position.
type Chunk struct {
	Text string
	Pos  int
}

// ChunkDocument splits content into overlapping chunks with break-at-boundary logic.
func ChunkDocument(content string, maxChars, overlapChars int) []Chunk {
	if maxChars <= 0 {
		maxChars = ChunkSizeChars
	}
	if overlapChars <= 0 {
		overlapChars = ChunkOverlapChars
	}
	if len(content) <= maxChars {
		return []Chunk{{Text: content, Pos: 0}}
	}
	var chunks []Chunk
	pos := 0
	for pos < len(content) {
		end := pos + maxChars
		if end > len(content) {
			end = len(content)
		}
		slice := content[pos:end]
		if end < len(content) {
			// Prefer break in last 30%
			searchStart := len(slice) * 7 / 10
			searchSlice := content[pos+searchStart : end]
			breakOff := findBreak(searchSlice)
			if breakOff >= 0 {
				end = pos + searchStart + breakOff
				slice = content[pos:end]
			}
		}
		if end <= pos {
			end = pos + maxChars
			if end > len(content) {
				end = len(content)
			}
			slice = content[pos:end]
		}
		chunks = append(chunks, Chunk{Text: slice, Pos: pos})
		if end >= len(content) {
			break
		}
		pos = end - overlapChars
		if pos <= chunks[len(chunks)-1].Pos {
			pos = end
		}
	}
	return chunks
}

func findBreak(s string) int {
	// Paragraph
	if i := lastIndex(s, "\n\n"); i >= 0 {
		return i + 2
	}
	// Sentence
	for _, sep := range []string{". ", ".\n", "? ", "?\n", "! ", "!\n"} {
		if i := lastIndex(s, sep); i >= 0 {
			return i + len(sep)
		}
	}
	// Line
	if i := lastIndex(s, "\n"); i >= 0 {
		return i + 1
	}
	// Word
	if i := lastIndex(s, " "); i >= 0 {
		return i + 1
	}
	return -1
}

func lastIndex(s, sep string) int {
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}
