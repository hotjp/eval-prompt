// Package bertscore provides BERTScore evaluation plugin.
// BERTScore: embedding-based text similarity metric using cosine similarity
// between candidate and reference token embeddings.
//
// Reference: Zhang et al., "BERTScore: Evaluating Text Generation with BERT", ICLR 2020
package bertscore

import (
	"context"
	"math"
	"math/rand"
	"strings"

	"github.com/eval-prompt/internal/service/eval"
	"github.com/eval-prompt/internal/service/eval/stats"
)

// Plugin implements eval.EvalPlugin for BERTScore evaluation.
type Plugin struct {
	embedder eval.Embedder
}

// NewPlugin creates a new BERTScore plugin with the given embedder.
func NewPlugin(embedder eval.Embedder) *Plugin {
	return &Plugin{embedder: embedder}
}

// Name implements eval.EvalPlugin.
func (p *Plugin) Name() string {
	return "bertscore"
}

// Description implements eval.EvalPlugin.
func (p *Plugin) Description() string {
	return "BERTScore: embedding-based text similarity metric"
}

// RequiredCapabilities implements eval.EvalPlugin.
func (p *Plugin) RequiredCapabilities() []string {
	return []string{"embedder"}
}

// Evaluate implements eval.EvalPlugin.
func (p *Plugin) Evaluate(ctx context.Context, input eval.EvalInput) eval.EvalResult {
	candidateTokens := tokenize(input.Candidate)
	referenceTokens := tokenize(input.Reference)

	// Get all unique tokens for embedding
	allTokens := uniqueTokens(candidateTokens, referenceTokens)
	if len(allTokens) == 0 {
		return eval.EvalResult{
			Score:   0.0,
			Details: map[string]any{"error": "no tokens to compare"},
		}
	}

	// Batch embed all tokens
	embeddings, err := p.embedder.Embed(ctx, allTokens)
	if err != nil {
		return eval.EvalResult{
			Score:   0.0,
			Details: map[string]any{"error": err.Error()},
		}
	}

	// Build token embedding maps
	candidateEmbeddings := make(map[string][]float64)
	referenceEmbeddings := make(map[string][]float64)
	for i, token := range allTokens {
		embedding := embeddings[i]
		for _, ct := range candidateTokens {
			if ct == token {
				candidateEmbeddings[ct] = embedding
			}
		}
		for _, rt := range referenceTokens {
			if rt == token {
				referenceEmbeddings[rt] = embedding
			}
		}
	}

	// Calculate R-Precision (reference-based) and P-Precision (candidate-based)
	// Precision: for each candidate token, max similarity with reference tokens
	// Recall: for each reference token, max similarity with candidate tokens

	var precisionSum, recallSum float64
	var candidateMatched, referenceMatched int

	// Compute precision: for each candidate token, find max cosine similarity with reference
	for _, ct := range candidateTokens {
		ctEmbed := candidateEmbeddings[ct]
		if ctEmbed == nil {
			continue
		}
		maxSim := -1.0
		for _, rt := range referenceTokens {
			rtEmbed := referenceEmbeddings[rt]
			if rtEmbed == nil {
				continue
			}
			sim := cosineSimilarity(ctEmbed, rtEmbed)
			if sim > maxSim {
				maxSim = sim
			}
		}
		if maxSim > 0 {
			precisionSum += maxSim
			candidateMatched++
		}
	}

	// Compute recall: for each reference token, find max cosine similarity with candidate
	for _, rt := range referenceTokens {
		rtEmbed := referenceEmbeddings[rt]
		if rtEmbed == nil {
			continue
		}
		maxSim := -1.0
		for _, ct := range candidateTokens {
			ctEmbed := candidateEmbeddings[ct]
			if ctEmbed == nil {
				continue
			}
			sim := cosineSimilarity(rtEmbed, ctEmbed)
			if sim > maxSim {
				maxSim = sim
			}
		}
		if maxSim > 0 {
			recallSum += maxSim
			referenceMatched++
		}
	}

	// Calculate Precision, Recall, F1
	var precision, recall, f1 float64
	if candidateMatched > 0 {
		precision = precisionSum / float64(len(candidateTokens))
	}
	if referenceMatched > 0 {
		recall = recallSum / float64(len(referenceTokens))
	}
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}

	// Bootstrap confidence interval for F1
	// Resample tokens and compute F1 for each resample
	bootstrapScores := make([]float64, 100)
	combinedTokens := make([]string, 0, len(candidateTokens)+len(referenceTokens))
	combinedTokens = append(combinedTokens, candidateTokens...)
	combinedTokens = append(combinedTokens, referenceTokens...)
	combinedLabels := make([]bool, len(combinedTokens)) // true = candidate
	for i := range candidateTokens {
		combinedLabels[i] = true
	}

	for i := 0; i < 100; i++ {
		// Bootstrap sample (sampling with replacement)
		n := len(combinedTokens)
		resampledCand := make([]string, 0, n)
		resampledRef := make([]string, 0, n)

		for j := 0; j < n; j++ {
			idx := rand.Intn(n)
			token := combinedTokens[idx]
			if combinedLabels[idx] {
				resampledCand = append(resampledCand, token)
			} else {
				resampledRef = append(resampledRef, token)
			}
		}

		// Compute F1 for resample (simplified: use original similarity matrix approach)
		// For bootstrap, we approximate by computing precision/recall on resampled sets
		var pSum, rSum float64
		var pMatched, rMatched int

		for _, ct := range resampledCand {
			ctEmbed := candidateEmbeddings[ct]
			if ctEmbed == nil {
				continue
			}
			maxSim := -1.0
			for _, rt := range resampledRef {
				rtEmbed := referenceEmbeddings[rt]
				if rtEmbed == nil {
					continue
				}
				sim := cosineSimilarity(ctEmbed, rtEmbed)
				if sim > maxSim {
					maxSim = sim
				}
			}
			if maxSim > 0 {
				pSum += maxSim
				pMatched++
			}
		}

		for _, rt := range resampledRef {
			rtEmbed := referenceEmbeddings[rt]
			if rtEmbed == nil {
				continue
			}
			maxSim := -1.0
			for _, ct := range resampledCand {
				ctEmbed := candidateEmbeddings[ct]
				if ctEmbed == nil {
					continue
				}
				sim := cosineSimilarity(rtEmbed, ctEmbed)
				if sim > maxSim {
					maxSim = sim
				}
			}
			if maxSim > 0 {
				rSum += maxSim
				rMatched++
			}
		}

		var p, r float64
		if pMatched > 0 {
			p = pSum / float64(len(resampledCand))
		}
		if rMatched > 0 {
			r = rSum / float64(len(resampledRef))
		}
		if p+r > 0 {
			bootstrapScores[i] = 2 * p * r / (p + r)
		}
	}

	low, high := stats.BootstrapCI(bootstrapScores, 0.95, 100)

	return eval.EvalResult{
		Score: f1,
		Dimensions: []eval.Dimension{
			{Name: "precision", Score: precision, Weight: 0.0},
			{Name: "recall", Score: recall, Weight: 0.0},
			{Name: "f1", Score: f1, Weight: 1.0},
		},
		ConfidenceInterval: &eval.ConfidenceInterval{
			Low:  low,
			High: high,
		},
		Details: map[string]any{
			"candidate_tokens": len(candidateTokens),
			"reference_tokens": len(referenceTokens),
			"embedder":         p.embedder.Name(),
		},
		Metadata: map[string]string{
			"paper": "Zhang et al., BERTScore: Evaluating Text Generation with BERT, ICLR 2020",
		},
	}
}

// tokenize splits text into tokens.
// Chinese text: character-level tokenization (BERT Chinese uses character-level).
// English text: whitespace tokenization.
func tokenize(text string) []string {
	var tokens []string
	var current strings.Builder
	isChinese := false

	for _, r := range text {
		if isChineseChar(r) {
			if current.Len() > 0 && !isChinese {
				// Flush English tokens
				tokens = append(tokens, strings.Fields(current.String())...)
				current.Reset()
			}
			// Flush Chinese character as separate token
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(r))
			isChinese = true
		} else if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			// Flush current buffer
			if current.Len() > 0 {
				if isChinese {
					tokens = append(tokens, current.String())
				} else {
					tokens = append(tokens, strings.Fields(current.String())...)
				}
				current.Reset()
			}
			isChinese = false
		} else {
			if current.Len() > 0 && isChinese {
				// Flush Chinese buffer
				tokens = append(tokens, current.String())
				current.Reset()
			}
			current.WriteRune(r)
			isChinese = false
		}
	}

	// Flush remaining
	if current.Len() > 0 {
		if isChinese {
			tokens = append(tokens, current.String())
		} else {
			tokens = append(tokens, strings.Fields(current.String())...)
		}
	}

	return tokens
}

// isChineseChar returns true if the rune is a Chinese character.
func isChineseChar(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF || // CJK Unified Ideographs
		r >= 0x3400 && r <= 0x4DBF || // CJK Unified Ideographs Extension A
		r >= 0x20000 && r <= 0x2A6DF // CJK Unified Ideographs Extension B
}

// uniqueTokens returns all unique tokens from both slices.
func uniqueTokens(tokens1, tokens2 []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, t := range tokens1 {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	for _, t := range tokens2 {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
