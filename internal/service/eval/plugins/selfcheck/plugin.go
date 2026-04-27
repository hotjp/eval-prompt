// Package selfcheck implements the SelfCheckGPT evaluation plugin.
//
// SelfCheckGPT (Manakul et al., ACL 2023) detects hallucinations in text
// by comparing original text with multiple LLM-generated samples.
// Low consistency between samples indicates possible hallucination.
// Score = 1 - hallucation_rate
//
// Reference:
// Manakul et al., "SelfCheckGPT: Zero-Resource Black-Box Hallucination Detection", ACL 2023
package selfcheck

import (
	"context"
	"strings"

	"github.com/eval-prompt/internal/service/eval"
	"github.com/eval-prompt/internal/service/eval/stats"
)

// Plugin implements eval.EvalPlugin for SelfCheckGPT evaluation.
type Plugin struct {
	judge eval.Judge
}

// NewPlugin creates a new SelfCheckGPT plugin with the given Judge.
func NewPlugin(judge eval.Judge) *Plugin {
	return &Plugin{judge: judge}
}

// Name implements eval.EvalPlugin.
func (p *Plugin) Name() string {
	return "selfcheck"
}

// Description implements eval.EvalPlugin.
func (p *Plugin) Description() string {
	return "SelfCheckGPT: hallucination detection via consistency checking"
}

// RequiredCapabilities implements eval.EvalPlugin.
func (p *Plugin) RequiredCapabilities() []string {
	return []string{"judge"}
}

// sentenceTruthPrompt checks if a sentence appears to be factual/truthful.
const sentenceTruthPrompt = `You are a factuality assessor. Given a sentence, assess whether it appears to be factual and truthful.

Sentence:
%s

Respond with only one word: "TRUE" if the sentence appears factual, or "FALSE" if it contains misinformation or cannot be verified as true.`

// paraphrasePrompt asks LLM to generate a paraphrase of the sentence.
const paraphrasePrompt = `Generate a paraphrase of this sentence preserving meaning but using different words. Output only the paraphrase:

%s`

// consistencyPrompt checks consistency between original and samples.
const consistencyPrompt = `Given an original sentence and alternative samples, assess consistency.

Original: %s

Samples:
%s

Rate the consistency on a scale of 0-10, where 10 means fully consistent and 0 means completely contradictory.
Respond with only the numeric score.`

// Evaluate implements eval.EvalPlugin.
// It evaluates hallucination by checking sentence-by-sentence truthfulness.
// For production use with actual SelfCheckGPT, provide pre-generated samples
// via Metadata["samples"].
func (p *Plugin) Evaluate(ctx context.Context, input eval.EvalInput) eval.EvalResult {
	// Extract sentences from candidate
	sentences := extractSentences(input.Candidate)
	if len(sentences) == 0 {
		return eval.EvalResult{
			Score:   1.0,
			Details: map[string]any{"message": "no sentences to evaluate"},
		}
	}

	// Get pre-generated samples from metadata if available
	var samples []string
	if samplesData, ok := input.Metadata["samples"]; ok {
		if samplesSlice, ok := samplesData.([]string); ok {
			samples = samplesSlice
		}
	}

	// Evaluate each sentence
	hallucinated := 0
	sentenceResults := make([]map[string]any, len(sentences))

	for i, sentence := range sentences {
		var hallucinatedFlag bool
		var consistencyScore float64 = 1.0

		if len(samples) > 0 {
			// Use pre-provided samples for consistency checking
			consistencyScore = checkConsistencyWithSamples(ctx, p.judge, sentence, samples)
			hallucinatedFlag = consistencyScore < 0.5
		} else {
			// Fallback: direct truth assessment
			hallucinatedFlag = !assessTruth(ctx, p.judge, sentence)
		}

		if hallucinatedFlag {
			hallucinated++
		}

		sentenceResults[i] = map[string]any{
			"sentence":          sentence,
			"hallucinated":       hallucinatedFlag,
			"consistency_score":  consistencyScore,
			"method":             getMethod(len(samples) > 0),
		}
	}

	// Calculate hallucination rate
	hallucinationRate := float64(hallucinated) / float64(len(sentences))
	score := 1.0 - hallucinationRate

	// Bootstrap confidence interval
	hallucinationFlags := make([]float64, len(sentences))
	for i, sr := range sentenceResults {
		if sr["hallucinated"].(bool) {
			hallucinationFlags[i] = 1.0
		} else {
			hallucinationFlags[i] = 0.0
		}
	}
	ciLow, ciHigh := stats.BootstrapCI(hallucinationFlags, 0.95, 100)

	return eval.EvalResult{
		Score: score,
		Dimensions: []eval.Dimension{
			{Name: "hallucinated_sentences", Score: float64(hallucinated), Weight: 0.0},
			{Name: "total_sentences", Score: float64(len(sentences)), Weight: 0.0},
			{Name: "hallucination_rate", Score: hallucinationRate, Weight: 0.0},
		},
		ConfidenceInterval: &eval.ConfidenceInterval{
			Low:  1.0 - ciHigh,
			High: 1.0 - ciLow,
		},
		Details: map[string]any{
			"sentences": sentenceResults,
			"method":    "selfcheck_consistency",
		},
		Metadata: map[string]string{
			"paper": "Manakul et al., SelfCheckGPT: Zero-Resource Black-Box Hallucination Detection, ACL 2023",
			"method": "consistency_checking",
		},
	}
}

// getMethod returns the evaluation method used based on sample availability.
func getMethod(hasSamples bool) string {
	if hasSamples {
		return "sample_consistency"
	}
	return "direct_assessment"
}

// extractSentences splits text into sentences.
func extractSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)

		// Check for sentence ending
		if r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？' {
			// Check it's not an abbreviation or decimal
			nextIsLower := false
			if i+1 < len(text) {
				nextIsLower = text[i+1] >= 'a' && text[i+1] <= 'z'
			}

			sentence := strings.TrimSpace(current.String())
			sentence = strings.TrimSuffix(sentence, ".")
			sentence = strings.TrimSuffix(sentence, "!")
			sentence = strings.TrimSuffix(sentence, "?")
			sentence = strings.TrimSuffix(sentence, "。")
			sentence = strings.TrimSuffix(sentence, "！")
			sentence = strings.TrimSuffix(sentence, "？")

			if len(sentence) > 5 && !nextIsLower {
				sentences = append(sentences, sentence)
			}
			current.Reset()
		}
	}

	// Don't forget the last sentence
	if current.Len() > 0 {
		sentence := strings.TrimSpace(current.String())
		if len(sentence) > 5 {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// assessTruth performs direct truth assessment on a sentence.
func assessTruth(ctx context.Context, judge eval.Judge, sentence string) bool {
	prompt := strings.Replace(sentenceTruthPrompt, "%s", sentence, 1)
	resp, err := judge.Score(ctx, prompt, sentence, "Truth assessment")
	if err != nil {
		return true // assume true on error
	}
	return resp >= 0.5
}

// checkConsistencyWithSamples checks consistency between sentence and provided samples.
func checkConsistencyWithSamples(ctx context.Context, judge eval.Judge, sentence string, samples []string) float64 {
	if len(samples) == 0 {
		return 0.5
	}

	var samplesText strings.Builder
	for i, s := range samples {
		samplesText.WriteString(s)
		if i < len(samples)-1 {
			samplesText.WriteString("\n")
		}
	}

	prompt := strings.Replace(consistencyPrompt, "%s", sentence, 1)
	prompt = strings.Replace(prompt, "%s", samplesText.String(), 1)

	resp, err := judge.Score(ctx, prompt, sentence, "Consistency check")
	if err != nil {
		return 0.5
	}

	return resp
}
