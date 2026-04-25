// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/eval-prompt/internal/service"
)

// TriggerHandler handles trigger API endpoints.
type TriggerHandler struct {
	triggerService service.TriggerServicer
	logger         *slog.Logger
}

// NewTriggerHandler creates a new TriggerHandler.
func NewTriggerHandler(triggerService service.TriggerServicer, logger *slog.Logger) *TriggerHandler {
	return &TriggerHandler{
		triggerService: triggerService,
		logger:         logger,
	}
}

// MatchTriggerRequest represents the request body for matching triggers.
type MatchTriggerRequest struct {
	Input string `json:"input"`
	Top   int    `json:"top,omitempty"`
}

// MatchTriggerResponse represents the response for matching triggers.
type MatchTriggerResponse struct {
	Matches []*service.MatchedPrompt `json:"matches"`
	Total   int                      `json:"total"`
}

// MatchTrigger godoc
// @Summary Match triggers
// @Description Find matching prompts for the given input
// @Tags trigger
// @Accept json
// @Produce json
// @Param body body MatchTriggerRequest true "Match trigger request"
// @Success 200 {object} MatchTriggerResponse
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/trigger/match [post]
func (h *TriggerHandler) MatchTrigger(w http.ResponseWriter, r *http.Request) {
	var req MatchTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.Input == "" {
		h.writeError(w, http.StatusBadRequest, "input is required")
		return
	}

	if req.Top <= 0 {
		req.Top = 5
	}

	matches, err := h.triggerService.MatchTrigger(r.Context(), req.Input, req.Top)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "match failed: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, MatchTriggerResponse{
		Matches: matches,
		Total:   len(matches),
	})
}

// ValidateAntiPatternsRequest represents the request body for validating anti-patterns.
type ValidateAntiPatternsRequest struct {
	Prompt string `json:"prompt"`
}

// ValidateAntiPatternsResponse represents the response for anti-pattern validation.
type ValidateAntiPatternsResponse struct {
	Valid      bool     `json:"valid"`
	Violations []string `json:"violations,omitempty"`
	Message    string   `json:"message,omitempty"`
}

// ValidateAntiPatterns godoc
// @Summary Validate anti-patterns
// @Description Check if a prompt violates any anti-patterns
// @Tags trigger
// @Accept json
// @Produce json
// @Param body body ValidateAntiPatternsRequest true "Validation request"
// @Success 200 {object} ValidateAntiPatternsResponse
// @Failure 400 {object} map[string]any
// @Router /api/v1/trigger/validate [post]
func (h *TriggerHandler) ValidateAntiPatterns(w http.ResponseWriter, r *http.Request) {
	var req ValidateAntiPatternsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.Prompt == "" {
		h.writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	err := h.triggerService.ValidateAntiPatterns(r.Context(), req.Prompt)
	if err != nil {
		// Anti-pattern violation found
		h.writeJSON(w, http.StatusOK, ValidateAntiPatternsResponse{
			Valid:   false,
			Message: err.Error(),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, ValidateAntiPatternsResponse{
		Valid:   true,
		Message: "prompt is valid",
	})
}

// InjectVariablesRequest represents the request body for injecting variables.
type InjectVariablesRequest struct {
	Prompt    string            `json:"prompt"`
	Variables map[string]string `json:"variables"`
}

// InjectVariablesResponse represents the response for variable injection.
type InjectVariablesResponse struct {
	Result string `json:"result"`
}

// InjectVariables godoc
// @Summary Inject variables
// @Description Replace variables in a prompt with provided values
// @Tags trigger
// @Accept json
// @Produce json
// @Param body body InjectVariablesRequest true "Variable injection request"
// @Success 200 {object} InjectVariablesResponse
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/trigger/inject [post]
func (h *TriggerHandler) InjectVariables(w http.ResponseWriter, r *http.Request) {
	var req InjectVariablesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.Prompt == "" {
		h.writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	if len(req.Variables) == 0 {
		h.writeError(w, http.StatusBadRequest, "variables are required")
		return
	}

	result, err := h.triggerService.InjectVariables(r.Context(), req.Prompt, req.Variables)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "injection failed: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, InjectVariablesResponse{
		Result: result,
	})
}

// GetAntiPatterns godoc
// @Summary Get anti-patterns
// @Description Get the list of defined anti-patterns
// @Tags trigger
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any
// @Router /api/v1/trigger/anti-patterns [get]
func (h *TriggerHandler) GetAntiPatterns(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]any{
		"anti_patterns": service.DefaultAntiPatterns,
	})
}

// writeJSON writes a JSON response.
func (h *TriggerHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *TriggerHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5", "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": fmt.Sprintf(format, args...),
	})
}
