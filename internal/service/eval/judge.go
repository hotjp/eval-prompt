package eval

import (
	"context"
	"strconv"
	"strings"
)

// Judge evaluates and scores text using LLM calls with Temperature=0.
type Judge interface {
	// Compare compares two outputs and returns a score.
	Compare(ctx context.Context, prompt, outputA, outputB string) (float64, error)

	// Score evaluates a single output against criteria and returns a score.
	Score(ctx context.Context, prompt, output, criteria string) (float64, error)

	// ScoreText evaluates a single output and returns both the score and the raw text response.
	// This is useful for tasks like fact extraction where the text content is needed.
	ScoreText(ctx context.Context, prompt, output, criteria string) (float64, string, error)
}

// LLMJudge wraps an LLMInvoker for judge evaluation with Temperature=0.
type LLMJudge struct {
	invoker LLMInvoker
	model   string
	// temperature for sampling (default 0, but G-Eval needs >0 for diverse sampling)
	temperature float64
}

// LLMInvoker is the interface for LLM calls (mirrors service.LLMInvoker).
type LLMInvoker interface {
	Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error)
}

// LLMResponse mirrors plugins/llm.LLMResponse.
type LLMResponse struct {
	Content    string
	Model      string
	TokensIn   int
	TokensOut  int
	StopReason string
}

// NewLLMJudge creates a new LLM judge with Temperature=0.
func NewLLMJudge(invoker LLMInvoker, model string) *LLMJudge {
	return &LLMJudge{
		invoker:    invoker,
		model:      model,
		temperature: 0,
	}
}

// NewLLMJudgeWithTemp creates a new LLM judge with custom temperature for sampling.
func NewLLMJudgeWithTemp(invoker LLMInvoker, model string, temperature float64) *LLMJudge {
	return &LLMJudge{
		invoker:    invoker,
		model:      model,
		temperature: temperature,
	}
}

// Compare implements Judge using pairwise comparison with Temperature=0.
func (j *LLMJudge) Compare(ctx context.Context, prompt, outputA, outputB string) (float64, error) {
	comparePrompt := prompt + "\n\nOutput A:\n" + outputA + "\n\nOutput B:\n" + outputB +
		"\n\nWhich output is better? Respond with only 'A' or 'B'."
	resp, err := j.invoker.Invoke(ctx, comparePrompt, j.model, 0)
	if err != nil {
		return 0, err
	}
	if resp.Content == "A" {
		return 1.0, nil
	} else if resp.Content == "B" {
		return 0.0, nil
	}
	return 0.5, nil // uncertain
}

// Score implements Judge using criteria-based evaluation with Temperature=0.
func (j *LLMJudge) Score(ctx context.Context, prompt, output, criteria string) (float64, error) {
	scorePrompt := prompt + "\n\nOutput to evaluate:\n" + output +
		"\n\nEvaluation criteria:\n" + criteria +
		"\n\nScore the output on a scale of 0-10. Respond with only the numeric score."
	resp, err := j.invoker.Invoke(ctx, scorePrompt, j.model, j.temperature)
	if err != nil {
		return 0, err
	}
	// Parse numeric score from response using standard library
	score, err := parseFloat(resp.Content)
	if err != nil {
		return 0.5, nil // fallback
	}
	return score / 10.0, nil // normalize to 0-1
}

// ScoreText implements Judge by returning both score and raw text response.
func (j *LLMJudge) ScoreText(ctx context.Context, prompt, output, criteria string) (float64, string, error) {
	scorePrompt := prompt + "\n\nOutput to evaluate:\n" + output +
		"\n\nEvaluation criteria:\n" + criteria +
		"\n\nScore the output on a scale of 0-10. Respond with only the numeric score."
	resp, err := j.invoker.Invoke(ctx, scorePrompt, j.model, j.temperature)
	if err != nil {
		return 0, "", err
	}
	// Parse numeric score from response
	score, err := parseFloat(resp.Content)
	if err != nil {
		return 0.5, resp.Content, nil // fallback
	}
	return score / 10.0, resp.Content, nil // normalize to 0-1
}

// parseFloat parses a float from a string, looking for the first number.
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	// Try to find a number at the start or after common prefixes
	s = strings.TrimLeft(s, " \t\n\r")

	// Handle "Score: 7.5" or "7.5/10" or just "7.5"
	// Look for first digit
	start := -1
	for i, ch := range s {
		if ch >= '0' && ch <= '9' {
			start = i
			break
		}
	}
	if start == -1 {
		return 0, strconv.ErrSyntax
	}

	// Extract the number portion
	end := start
	hasDecimal := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if ch >= '0' && ch <= '9' {
			end = i + 1
		} else if ch == '.' && !hasDecimal {
			hasDecimal = true
			end = i + 1
		} else if ch == '/' {
			break
		} else if ch != ' ' && ch != '\t' && (ch < '0' || ch > '9') && ch != '.' {
			if i > start {
				end = i
			}
			break
		}
	}

	numStr := s[start:end]
	if numStr == "" || numStr == "." {
		return 0, strconv.ErrSyntax
	}

	return strconv.ParseFloat(numStr, 64)
}
