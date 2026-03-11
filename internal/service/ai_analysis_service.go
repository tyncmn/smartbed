// Package service – AI Analysis Service.
// Calls OpenAI Chat Completions with patient vitals + profile context
// to produce a structured sleep and health assessment.
// Requires OPENAI_API_KEY environment variable.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const openAIEndpoint = "https://api.openai.com/v1/chat/completions"

// AIAnalysisService calls OpenAI to assess patient health state.
type AIAnalysisService struct {
	db     *sqlx.DB
	apiKey string
	model  string
}

// NewAIAnalysisService creates a new AIAnalysisService.
// apiKey must be set from OPENAI_API_KEY env var — never hardcoded.
func NewAIAnalysisService(db *sqlx.DB, apiKey, model string) *AIAnalysisService {
	return &AIAnalysisService{db: db, apiKey: apiKey, model: model}
}

// ─── Output DTOs ──────────────────────────────────────────────────────────────

// AIHealthAlert is a single AI-detected health concern.
type AIHealthAlert struct {
	Severity string `json:"severity"` // "info" | "warning" | "critical"
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

// AIMedicineHint is an over-the-counter suggestion.
type AIMedicineHint struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
	Note    string `json:"note"` // always includes: "consult a doctor or pharmacist before use"
}

// AIAnalysisResult is the structured output returned from OpenAI.
type AIAnalysisResult struct {
	SleepState           string           `json:"sleep_state"`    // deep | light | rem | awake | disturbed | unknown
	OverallStatus        string           `json:"overall_status"` // ok | warning | critical
	Analysis             string           `json:"analysis"`
	HealthAlerts         []AIHealthAlert  `json:"health_alerts"`
	EmergencyActions     []string         `json:"emergency_actions"`
	MedicineSuggestions  []AIMedicineHint `json:"medicine_suggestions"`
	LifestyleSuggestions []string         `json:"lifestyle_suggestions"`
	Disclaimer           string           `json:"disclaimer"`
}

// ─── Internal OpenAI wire types ───────────────────────────────────────────────

type openAIRequest struct {
	Model          string          `json:"model"`
	Messages       []openAIMessage `json:"messages"`
	ResponseFormat openAIFormat    `json:"response_format"`
	Temperature    float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIFormat struct {
	Type string `json:"type"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ─── Core Analysis ────────────────────────────────────────────────────────────

// AnalyzeInput holds all pre-fetched context for the AI call.
type AnalyzeInput struct {
	UserID          uuid.UUID
	Age             int
	Sex             string
	Conditions      []string
	Summary         *SleepSummary
	Timeline        []DailyTimelineEntry
	RecentAlertMsgs []string
}

// Analyze builds a prompt from the given context and calls OpenAI.
func (s *AIAnalysisService) Analyze(ctx context.Context, input AnalyzeInput) (*AIAnalysisResult, error) {
	currentHour := time.Now().UTC().Hour()
	timeCtx := "daytime"
	switch {
	case currentHour >= 22 || currentHour < 6:
		timeCtx = "night (sleep period)"
	case currentHour < 12:
		timeCtx = "morning"
	case currentHour < 18:
		timeCtx = "afternoon"
	default:
		timeCtx = "evening"
	}

	ageStr := "unknown"
	if input.Age > 0 {
		ageStr = fmt.Sprintf("%d years old", input.Age)
	}
	conditions := "none reported"
	if len(input.Conditions) > 0 {
		conditions = ""
		for i, c := range input.Conditions {
			if i > 0 {
				conditions += ", "
			}
			conditions += c
		}
	}

	summaryText := "no sleep session data available"
	if input.Summary != nil && input.Summary.TotalNights > 0 {
		summaryText = fmt.Sprintf(
			"avg quality score: %.1f/100, avg duration: %.0f min, total nights: %d, disturbed nights: %d (over %d days)",
			input.Summary.AvgQualityScore, input.Summary.AvgDurationMins,
			input.Summary.TotalNights, input.Summary.DisturbedNights, input.Summary.PeriodDays,
		)
	}

	timelineText := "no nightly data available"
	if len(input.Timeline) > 0 {
		timelineText = ""
		for _, t := range input.Timeline {
			disturbed := ""
			if t.IsDisturbed {
				disturbed = " [DISTURBED]"
			}
			timelineText += fmt.Sprintf("  %s: quality=%.0f, duration=%.0f min, disturbances=%d%s\n",
				t.Date, t.QualityScore, t.DurationMins, t.DisturbanceCount, disturbed)
		}
	}

	alertsText := "no recent alerts"
	if len(input.RecentAlertMsgs) > 0 {
		alertsText = ""
		for _, a := range input.RecentAlertMsgs {
			alertsText += "  - " + a + "\n"
		}
	}

	systemPrompt := `You are a clinical AI assistant embedded in SmartBed, a sleep and health monitoring platform.
Analyze the patient data and produce a structured JSON health assessment.

Respond with ONLY valid JSON — no markdown, no explanations outside the JSON.

Required JSON schema:
{
  "sleep_state": "<deep_sleep | light_sleep | rem | awake | disturbed | unknown>",
  "overall_status": "<ok | warning | critical>",
  "analysis": "<2–4 sentence plain-English summary of the patient's current sleep and health state>",
  "health_alerts": [
    { "severity": "<info|warning|critical>", "title": "<short title>", "detail": "<clinical explanation>" }
  ],
  "emergency_actions": ["<action>"],
  "medicine_suggestions": [
    { "name": "<OTC name>", "purpose": "<why it may help>", "note": "consult a doctor or pharmacist before use" }
  ],
  "lifestyle_suggestions": ["<practical suggestion>"]
}

Rules:
- If overall_status is "ok": emergency_actions MUST be an empty array.
- If SpO2 < 90% appears in alerts: overall_status MUST be "critical" and emergency_actions MUST include "Call emergency services immediately".
- Only suggest over-the-counter medications. Every medicine note MUST include "consult a doctor or pharmacist before use".
- Speak in terms of observations and risk levels. Do not make definitive diagnoses.`

	userPrompt := fmt.Sprintf(`Patient:
- Age: %s | Sex: %s | Existing conditions: %s
- Current time context: %s (UTC hour %d)

Sleep summary (%dd window):
  %s

Nightly timeline:
%s

Recent health alerts:
%s

Return your assessment as JSON.`,
		ageStr, input.Sex, conditions,
		timeCtx, currentHour,
		input.Summary.PeriodDays,
		summaryText,
		timelineText,
		alertsText,
	)

	reqBody := openAIRequest{
		Model: s.model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: openAIFormat{Type: "json_object"},
		Temperature:    0.2,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai call: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	var oaResp openAIResponse
	if err := json.Unmarshal(respBytes, &oaResp); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}
	if oaResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s", oaResp.Error.Message)
	}
	if len(oaResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	var result AIAnalysisResult
	if err := json.Unmarshal([]byte(oaResp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("parse ai json: %w", err)
	}

	result.Disclaimer = "This AI analysis is for informational purposes only and does not constitute medical advice. Always consult a qualified healthcare professional."

	// Ensure slices are never nil
	if result.HealthAlerts == nil {
		result.HealthAlerts = []AIHealthAlert{}
	}
	if result.EmergencyActions == nil {
		result.EmergencyActions = []string{}
	}
	if result.MedicineSuggestions == nil {
		result.MedicineSuggestions = []AIMedicineHint{}
	}
	if result.LifestyleSuggestions == nil {
		result.LifestyleSuggestions = []string{}
	}

	return &result, nil
}
