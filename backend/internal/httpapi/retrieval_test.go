package httpapi

import (
	"math"
	"strings"
	"testing"
)

func TestChunkExtractedTextUsesBoundedOverlappingChunks(t *testing.T) {
	text := strings.Repeat("alpha beta gamma delta ", 300)
	chunks := chunkExtractedText(text)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for index, chunk := range chunks {
		if chunk.Index != index || chunk.CharCount == 0 || chunk.CharCount > fileChunkMaxRunes || chunk.TokenEstimate < 1 {
			t.Fatalf("invalid chunk %d: %+v", index, chunk)
		}
		if len(chunk.Embedding) != localEmbeddingSize {
			t.Fatalf("embedding size=%d", len(chunk.Embedding))
		}
	}
	firstTail := []rune(chunks[0].Content)
	second := chunks[1].Content
	overlapProbe := strings.TrimSpace(string(firstTail[max(0, len(firstTail)-100):]))
	if overlapProbe != "" && !strings.Contains(second, overlapProbe) {
		t.Fatalf("expected chunk overlap; probe=%q", overlapProbe)
	}
}

func TestLocalEmbeddingRoundTripAndSimilarity(t *testing.T) {
	revenue := localTextEmbedding("quarterly revenue increased by forty percent")
	similar := localTextEmbedding("what was the quarterly revenue increase")
	unrelated := localTextEmbedding("carrot soup cooking recipe")
	if cosineSimilarity(revenue, similar) <= cosineSimilarity(revenue, unrelated) {
		t.Fatal("relevant text should have greater similarity")
	}
	decoded := decodeEmbedding(encodeEmbedding(revenue))
	if len(decoded) != localEmbeddingSize || math.Abs(cosineSimilarity(revenue, decoded)-1) > 1e-9 {
		t.Fatal("embedding round trip changed the vector")
	}
	if decodeEmbedding(`{"invalid":true}`) != nil {
		t.Fatal("invalid embedding must be rejected")
	}
}
