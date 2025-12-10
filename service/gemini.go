package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tasklineby/certify-backend/entity"
)

const (
	geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

	// DocumentComparisonPrompt is the prompt sent to Gemini for document comparison
	DocumentComparisonPrompt = `You are a document verification expert. Compare the ORIGINAL document (first) with the PROVIDED document (second).

ALWAYS provide a detailed response. Analyze:
- Text differences (names, dates, numbers, addresses)
- Visual differences (logos, signatures, stamps, layout)
- Signs of tampering or forgery

Return this JSON structure:
{
  "score": 0.0-1.0,
  "is_authentic": true/false,
  "confidence": "low|medium|high",
  "differences": [
    {"location": "specific area", "original_value": "what original shows", "provided_value": "what provided shows", "severity": "minor|moderate|critical", "description": "explain the difference"}
  ],
  "findings": [
    {"category": "text|layout|visual|tampering", "description": "what you found", "severity": "info|warning|critical"}
  ],
  "summary": "2-3 sentences explaining overall comparison result and recommendation"
}

IMPORTANT:
- Score 0.95-1.0 = identical, 0.85-0.94 = minor diffs, 0.70-0.84 = moderate, <0.70 = major issues
- ALWAYS include at least 1 finding explaining your analysis
- ALWAYS write a summary even if documents match
- If documents differ, list specific differences with exact values`
)

// GeminiClient handles communication with Gemini API for document analysis
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// GeminiComparisonResponse represents the structured response from Gemini
type GeminiComparisonResponse struct {
	Score       float64            `json:"score"`
	IsAuthentic bool               `json:"is_authentic"`
	Confidence  string             `json:"confidence"`
	Differences []GeminiDifference `json:"differences"`
	Findings    []GeminiFinding    `json:"findings"`
	Summary     string             `json:"summary"`
}

// GeminiDifference represents a specific difference found between documents
type GeminiDifference struct {
	Location      string `json:"location"`
	OriginalValue string `json:"original_value"`
	ProvidedValue string `json:"provided_value"`
	Severity      string `json:"severity"`
	Description   string `json:"description"`
}

// GeminiFinding represents a single finding from document analysis
type GeminiFinding struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// GeminiRequest represents the request structure for Gemini API
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
}

// GeminiContent represents content in Gemini request
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of content (text or inline data)
type GeminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inlineData,omitempty"`
}

// GeminiInlineData represents inline binary data (images/PDFs)
type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

// GeminiGenerationConfig represents generation configuration
type GeminiGenerationConfig struct {
	Temperature      float64 `json:"temperature"`
	MaxOutputTokens  int     `json:"maxOutputTokens"`
	ResponseMimeType string  `json:"responseMimeType"`
}

// GeminiAPIResponse represents the response from Gemini API
type GeminiAPIResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(apiKey, model string) *GeminiClient {
	return &GeminiClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

// CompareDocumentsWithPhotos compares original PDF with photos using Gemini
func (c *GeminiClient) CompareDocumentsWithPhotos(ctx context.Context, originalPDF []byte, photos [][]byte) (*entity.DocumentAnalysisResult, *GeminiComparisonResponse, error) {
	parts := []GeminiPart{
		{Text: DocumentComparisonPrompt},
		{
			InlineData: &GeminiInlineData{
				MimeType: "application/pdf",
				Data:     base64.StdEncoding.EncodeToString(originalPDF),
			},
		},
	}

	// Add each photo
	for _, photo := range photos {
		mimeType := detectImageMimeType(photo)
		parts = append(parts, GeminiPart{
			InlineData: &GeminiInlineData{
				MimeType: mimeType,
				Data:     base64.StdEncoding.EncodeToString(photo),
			},
		})
	}

	return c.sendRequest(ctx, parts)
}

// CompareDocumentsWithPDF compares original PDF with another PDF using Gemini
func (c *GeminiClient) CompareDocumentsWithPDF(ctx context.Context, originalPDF []byte, comparisonPDF []byte) (*entity.DocumentAnalysisResult, *GeminiComparisonResponse, error) {
	parts := []GeminiPart{
		{Text: DocumentComparisonPrompt},
		{
			InlineData: &GeminiInlineData{
				MimeType: "application/pdf",
				Data:     base64.StdEncoding.EncodeToString(originalPDF),
			},
		},
		{
			InlineData: &GeminiInlineData{
				MimeType: "application/pdf",
				Data:     base64.StdEncoding.EncodeToString(comparisonPDF),
			},
		},
	}

	return c.sendRequest(ctx, parts)
}

// sendRequest sends the request to Gemini API and parses the response
func (c *GeminiClient) sendRequest(ctx context.Context, parts []GeminiPart) (*entity.DocumentAnalysisResult, *GeminiComparisonResponse, error) {
	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: parts},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:      0.1,
			MaxOutputTokens:  2048,
			ResponseMimeType: "application/json",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf(geminiAPIURL, c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Gemini API error", "status", resp.StatusCode, "body", string(body))
		return nil, nil, fmt.Errorf("Gemini API error: status %d", resp.StatusCode)
	}

	var apiResp GeminiAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, nil, fmt.Errorf("Gemini API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return nil, nil, fmt.Errorf("empty response from Gemini API")
	}

	// Parse the JSON response from Gemini
	var sb strings.Builder
	for _, part := range apiResp.Candidates[0].Content.Parts {
		sb.WriteString(part.Text)
	}

	responseText := normalizeGeminiResponse(sb.String())

	var comparisonResp GeminiComparisonResponse
	if err := parseGeminiJSON(responseText, &comparisonResp); err != nil {
		slog.Error("Failed to parse Gemini response as JSON", "response", responseText, "err", err)
		return nil, nil, fmt.Errorf("failed to parse comparison response: %w", err)
	}

	// Convert differences to entity format
	var differences []entity.DocumentDifference
	for _, d := range comparisonResp.Differences {
		differences = append(differences, entity.DocumentDifference{
			Location:      d.Location,
			OriginalValue: d.OriginalValue,
			ProvidedValue: d.ProvidedValue,
			Severity:      d.Severity,
			Description:   d.Description,
		})
	}

	// Convert findings to entity format
	var findings []entity.AnalysisFinding
	for _, f := range comparisonResp.Findings {
		findings = append(findings, entity.AnalysisFinding{
			Category:    f.Category,
			Description: f.Description,
			Severity:    f.Severity,
		})
	}

	// Convert to entity.DocumentAnalysisResult
	result := &entity.DocumentAnalysisResult{
		Score:       comparisonResp.Score,
		IsAuthentic: comparisonResp.IsAuthentic,
		Confidence:  comparisonResp.Confidence,
		Differences: differences,
		Findings:    findings,
		Summary:     comparisonResp.Summary,
	}

	slog.Info("Gemini document comparison completed",
		"score", comparisonResp.Score,
		"is_authentic", comparisonResp.IsAuthentic,
		"confidence", comparisonResp.Confidence,
		"differences_count", len(comparisonResp.Differences),
		"findings_count", len(comparisonResp.Findings))

	return result, &comparisonResp, nil
}

// detectImageMimeType detects the MIME type of an image based on magic bytes
func detectImageMimeType(data []byte) string {
	if len(data) < 4 {
		return "image/jpeg" // default
	}

	// Check magic bytes
	switch {
	case data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image/jpeg"
	case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
		return "image/png"
	case data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46:
		return "image/gif"
	case data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46:
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// normalizeGeminiResponse removes common formatting Gemini may add (e.g., code fences)
// and trims whitespace so the JSON decoder receives a clean payload.
func normalizeGeminiResponse(resp string) string {
	resp = strings.TrimSpace(resp)

	// Handle code fences ```json ... ```
	if strings.HasPrefix(resp, "```") {
		// Remove first fence line
		if idx := strings.Index(resp, "\n"); idx != -1 {
			resp = resp[idx+1:]
		}
		resp = strings.TrimPrefix(resp, "json")
		resp = strings.TrimPrefix(resp, "JSON")
		resp = strings.TrimSpace(resp)
		// Remove trailing fence if present
		resp = strings.TrimSuffix(resp, "```")
		resp = strings.TrimSpace(resp)
	}

	return resp
}

// parseGeminiJSON attempts to unmarshal JSON and falls back to repairing
// truncated JSON by closing unclosed brackets/braces.
func parseGeminiJSON(responseText string, out any) error {
	if err := json.Unmarshal([]byte(responseText), out); err == nil {
		return nil
	}

	// Try to repair truncated JSON by closing open structures
	repaired := repairTruncatedJSON(responseText)
	if err := json.Unmarshal([]byte(repaired), out); err == nil {
		slog.Warn("Parsed repaired JSON response", "original_len", len(responseText), "repaired_len", len(repaired))
		return nil
	}

	return fmt.Errorf("invalid JSON response")
}

// repairTruncatedJSON attempts to fix truncated JSON by:
// 1. Finding the last complete JSON element
// 2. Closing any unclosed brackets/braces
func repairTruncatedJSON(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return "{}"
	}

	// Find position of last complete value (after a complete string, number, bool, null, }, or ])
	runes := []rune(s)
	inString := false
	escaped := false
	lastCompleteValue := 0

	for i, r := range runes {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			if !inString {
				// Just completed a string
				lastCompleteValue = i + 1
			}
			continue
		}
		if !inString {
			switch r {
			case '}', ']':
				lastCompleteValue = i + 1
			case ',':
				// Comma after a value means previous value was complete
				lastCompleteValue = i
			}
		}
	}

	// If we're inside a string or have trailing incomplete content, truncate
	if inString || lastCompleteValue < len(runes) {
		// Check what's after lastCompleteValue
		trailing := strings.TrimSpace(string(runes[lastCompleteValue:]))

		// If trailing is just structural chars that need values, truncate
		if trailing != "" && !strings.HasPrefix(trailing, "}") && !strings.HasPrefix(trailing, "]") {
			s = string(runes[:lastCompleteValue])
		}
	}

	// Remove trailing incomplete elements: comma, colon, or incomplete key
	s = strings.TrimSpace(s)
	for {
		trimmed := false
		// Remove trailing comma
		if strings.HasSuffix(s, ",") {
			s = strings.TrimSuffix(s, ",")
			s = strings.TrimSpace(s)
			trimmed = true
		}
		// Remove trailing colon (incomplete key-value)
		if strings.HasSuffix(s, ":") {
			// Find and remove the key too
			s = strings.TrimSuffix(s, ":")
			s = strings.TrimSpace(s)
			// Remove the key string
			if strings.HasSuffix(s, "\"") {
				idx := strings.LastIndex(s[:len(s)-1], "\"")
				if idx >= 0 {
					s = strings.TrimSpace(s[:idx])
				}
			}
			// Remove comma before the removed key if present
			s = strings.TrimSuffix(s, ",")
			s = strings.TrimSpace(s)
			trimmed = true
		}
		if !trimmed {
			break
		}
	}

	// Now close any unclosed brackets/braces
	var stack []rune
	inString = false
	escaped = false

	for _, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if !inString {
			switch r {
			case '{':
				stack = append(stack, '}')
			case '[':
				stack = append(stack, ']')
			case '}', ']':
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}

	// Close unclosed structures in reverse order
	for i := len(stack) - 1; i >= 0; i-- {
		s += string(stack[i])
	}

	return s
}
