package codeindex

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Chunk is a line-bounded slice of file content.
type Chunk struct {
	StartLine   int
	EndLine     int
	Text        string
	ContentHash string
}

// ChunkLines splits content into overlapping line blocks.
func ChunkLines(content []byte, chunkLines, overlap int) []Chunk {
	if chunkLines <= 0 {
		chunkLines = 80
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkLines {
		overlap = chunkLines / 4
	}

	text := string(content)
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}

	var chunks []Chunk
	step := chunkLines - overlap
	if step <= 0 {
		step = chunkLines
	}

	for start := 0; start < len(lines); start += step {
		end := start + chunkLines
		if end > len(lines) {
			end = len(lines)
		}

		block := strings.Join(lines[start:end], "\n")
		sum := sha256.Sum256([]byte(block))
		chunks = append(chunks, Chunk{
			StartLine:   start + 1,
			EndLine:     end,
			Text:        block,
			ContentHash: hex.EncodeToString(sum[:]),
		})

		if end >= len(lines) {
			break
		}
	}

	return chunks
}
