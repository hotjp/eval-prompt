// Package eval provides the evaluation orchestrator and plugin system.
//
// custom_rubric.go implements a YAML-based custom rubric parser that dynamically
// generates EvalPlugin instances. Rubrics are defined in YAML with dimensions,
// weights, and criteria. Each parsed rubric is automatically registered as
// "custom:{rubric_name}" for use in evaluation pipelines.
//
// YAML format:
//
//	name: my-rubric
//	description: Custom evaluation rubric
//	dimensions:
//	  - name: accuracy
//	    weight: 0.5
//	    criteria: |
//	      Evaluate the factual accuracy of the response.
//	  - name: helpfulness
//	    weight: 0.5
//	    criteria: |
//	      Evaluate how helpful and relevant the response is.
package eval

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// CustomRubric represents a user-defined evaluation rubric.
type CustomRubric struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Dimensions  []RubricDimension `yaml:"dimensions"`
}

// RubricDimension defines a single evaluation dimension with weight and criteria.
type RubricDimension struct {
	Name     string  `yaml:"name"`
	Weight   float64 `yaml:"weight"`
	Criteria string  `yaml:"criteria"`
}

// customRubricPlugin implements EvalPlugin for a parsed CustomRubric.
type customRubricPlugin struct {
	rubric CustomRubric
	judge  Judge
}

// Name returns the plugin name in "custom:{rubric_name}" format.
func (p *customRubricPlugin) Name() string {
	return "custom:" + p.rubric.Name
}

// Description returns the rubric description.
func (p *customRubricPlugin) Description() string {
	return p.rubric.Description
}

// RequiredCapabilities returns the capabilities required to run this plugin.
func (p *customRubricPlugin) RequiredCapabilities() []string {
	return []string{"llm"}
}

// Evaluate runs the evaluation using the custom rubric dimensions.
func (p *customRubricPlugin) Evaluate(ctx context.Context, input EvalInput) EvalResult {
	result := EvalResult{
		Score:      0.0,
		Dimensions: make([]Dimension, 0, len(p.rubric.Dimensions)),
		Details:    make(map[string]any),
		Metadata:   make(map[string]string),
	}

	if len(p.rubric.Dimensions) == 0 {
		result.Metadata["error"] = "no dimensions defined in rubric"
		return result
	}

	// Calculate total weight for normalization
	var totalWeight float64
	for _, dim := range p.rubric.Dimensions {
		totalWeight += dim.Weight
	}

	if totalWeight <= 0 {
		totalWeight = 1.0 // avoid division by zero
	}

	var weightedSum float64

	for _, dim := range p.rubric.Dimensions {
		dimScore := p.evaluateDimension(ctx, input, dim)
		normalizedWeight := dim.Weight / totalWeight
		weightedSum += dimScore * normalizedWeight

		result.Dimensions = append(result.Dimensions, Dimension{
			Name:   dim.Name,
			Score:  dimScore,
			Weight: dim.Weight,
		})
	}

	result.Score = weightedSum
	result.Metadata["rubric_name"] = p.rubric.Name
	result.Metadata["dimension_count"] = fmt.Sprintf("%d", len(p.rubric.Dimensions))

	return result
}

// evaluateDimension evaluates a single dimension using the configured judge.
func (p *customRubricPlugin) evaluateDimension(ctx context.Context, input EvalInput, dim RubricDimension) float64 {
	// If no judge is configured, use basic keyword matching as fallback
	if p.judge == nil {
		return p.basicEvaluate(input.Candidate, dim.Criteria)
	}

	// Use LLM judge for criteria-based scoring
	score, err := p.judge.Score(ctx, "", input.Candidate, dim.Criteria)
	if err != nil {
		return 0.5 // fallback on error
	}
	return score
}

// basicEvaluate provides a simple fallback evaluation using keyword matching.
func (p *customRubricPlugin) basicEvaluate(candidate, criteria string) float64 {
	// Simple heuristic: count matching keywords between criteria and candidate
	criteriaWords := strings.Fields(strings.ToLower(criteria))
	candidateLower := strings.ToLower(candidate)

	var matches int
	for _, word := range criteriaWords {
		if len(word) > 3 && strings.Contains(candidateLower, word) {
			matches++
		}
	}

	if len(criteriaWords) == 0 {
		return 0.5
	}

	// Normalize to 0-1 range
	ratio := float64(matches) / float64(len(criteriaWords))
	// Apply sigmoid-like transformation to avoid extreme scores
	score := ratio / (ratio + 0.5)
	return score
}

// ParseYAMLFile parses a CustomRubric from a YAML file.
func ParseYAMLFile(filePath string) (*CustomRubric, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("eval: failed to read rubric file: %w", err)
	}

	return ParseYAML(data)
}

// ParseYAML parses a CustomRubric from YAML content.
func ParseYAML(data []byte) (*CustomRubric, error) {
	var rubric CustomRubric
	if err := yaml.Unmarshal(data, &rubric); err != nil {
		return nil, fmt.Errorf("eval: failed to parse rubric YAML: %w", err)
	}

	if err := rubric.validate(); err != nil {
		return nil, fmt.Errorf("eval: invalid rubric: %w", err)
	}

	return &rubric, nil
}

// validate checks the rubric for required fields and valid values.
func (r *CustomRubric) validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Dimensions) == 0 {
		return fmt.Errorf("at least one dimension is required")
	}

	var totalWeight float64
	for i, dim := range r.Dimensions {
		if dim.Name == "" {
			return fmt.Errorf("dimension %d: name is required", i)
		}
		if dim.Weight < 0 {
			return fmt.Errorf("dimension %q: weight cannot be negative", dim.Name)
		}
		totalWeight += dim.Weight
	}

	if totalWeight == 0 {
		return fmt.Errorf("total weight of all dimensions must be greater than 0")
	}

	return nil
}

// GeneratePlugin creates an EvalPlugin from the rubric, optionally using a Judge.
func (r *CustomRubric) GeneratePlugin(judge Judge) EvalPlugin {
	return &customRubricPlugin{
		rubric: *r,
		judge:  judge,
	}
}

// RegisterFromYAML parses a YAML file and registers the resulting plugin.
func RegisterFromYAML(filePath string, judge Judge) error {
	rubric, err := ParseYAMLFile(filePath)
	if err != nil {
		return err
	}

	plugin := rubric.GeneratePlugin(judge)
	Register(plugin)

	return nil
}

// RegisterWithJudge registers a rubric from YAML data with a specific judge.
func RegisterWithJudge(data []byte, judge Judge) error {
	rubric, err := ParseYAML(data)
	if err != nil {
		return err
	}

	plugin := rubric.GeneratePlugin(judge)
	Register(plugin)

	return nil
}
