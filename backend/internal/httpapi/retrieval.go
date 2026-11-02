package httpapi

import (
	"encoding/json"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

const (
	fileChunkMaxRunes     = 2000
	fileChunkOverlapRunes = 200
	localEmbeddingModel   = "local-hash-v1"
	localEmbeddingSize    = 64
)

type fileChunk struct {
	Index         int
	Content       string
	CharCount     int
	TokenEstimate int
	Embedding     []float64
}

func chunkExtractedText(value string) []fileChunk {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) == 0 {
		return nil
	}
	chunks := []fileChunk{}
	for start := 0; start < len(runes); {
		end := min(start+fileChunkMaxRunes, len(runes))
		if end < len(runes) {
			floor := start + fileChunkMaxRunes/2
			for boundary := end; boundary > floor; boundary-- {
				if unicode.IsSpace(runes[boundary-1]) {
					end = boundary
					break
				}
			}
		}
		content := strings.TrimSpace(string(runes[start:end]))
		if content != "" {
			chunks = append(chunks, fileChunk{
				Index: len(chunks), Content: content, CharCount: len([]rune(content)),
				TokenEstimate: max(1, (len([]rune(content))+3)/4), Embedding: localTextEmbedding(content),
			})
		}
		if end >= len(runes) {
			break
		}
		next := end - fileChunkOverlapRunes
		if next <= start {
			next = end
		}
		start = next
	}
	return chunks
}

func localTextEmbedding(value string) []float64 {
	vector := make([]float64, localEmbeddingSize)
	tokens := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for _, token := range tokens {
		if token == "" {
			continue
		}
		hash := fnv.New64a()
		_, _ = hash.Write([]byte(token))
		value := hash.Sum64()
		index := int(value % localEmbeddingSize)
		sign := 1.0
		if value&(1<<63) != 0 {
			sign = -1
		}
		vector[index] += sign
	}
	var magnitude float64
	for _, value := range vector {
		magnitude += value * value
	}
	if magnitude == 0 {
		return vector
	}
	magnitude = math.Sqrt(magnitude)
	for index := range vector {
		vector[index] = vector[index] / magnitude
	}
	return vector
}

func encodeEmbedding(vector []float64) string {
	raw, _ := json.Marshal(vector)
	return string(raw)
}

func decodeEmbedding(value string) []float64 {
	var vector []float64
	if json.Unmarshal([]byte(value), &vector) != nil || len(vector) != localEmbeddingSize {
		return nil
	}
	return vector
}

func cosineSimilarity(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	var score float64
	for index := range left {
		score += left[index] * right[index]
	}
	return score
}
