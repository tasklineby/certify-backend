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

	"github.com/tasklineby/certify-backend/entity"
)

const (
	geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

	// DocumentComparisonPrompt is the prompt sent to Gemini for document comparison
	DocumentComparisonPrompt = `You are a document verification expert specializing in detecting differences and potential forgeries. Compare the provided documents with extreme attention to detail.

TASK: Compare the ORIGINAL document with the PROVIDED document(s) and identify ALL differences.

ANALYZE THESE ASPECTS AND HIGHLIGHT SPECIFIC DIFFERENCES:

1. **TEXT CONTENT**
   - Compare every text field, number, date, name, address
   - Note exact differences: "Field X shows 'ABC' in original but 'ABD' in provided"
   - Flag missing or added text sections

2. **LAYOUT & FORMATTING**
   - Document structure, margins, spacing
   - Font types, sizes, colors
   - Table structures and cell positions

3. **VISUAL ELEMENTS**
   - Logos: position, size, clarity, color accuracy
   - Signatures: presence, position, appearance
   - Stamps/seals: placement, clarity, authenticity markers
   - Watermarks and security features

4. **TAMPERING INDICATORS**
   - Pixel inconsistencies, blur patterns around text/images
   - Misaligned elements, uneven lighting/shadows
   - Copy-paste artifacts, font inconsistencies
   - EXIF data anomalies (if detectable)

5. **QUALITY ASSESSMENT**
   - Image resolution comparison
   - Scanning artifacts vs digital manipulation
   - Color profile consistency

RESPOND ONLY WITH VALID JSON in the following format:
{
  "score": <float between 0.0 and 1.0>,
  "is_authentic": <boolean>,
  "confidence": "<low|medium|high>",
  "differences": [
    {
      "location": "<specific location in document, e.g. 'Header section', 'Bottom-right signature area', 'Line 3, Column 2'>",
      "original_value": "<what appears in original document>",
      "provided_value": "<what appears in provided document>",
      "severity": "<minor|moderate|critical>",
      "description": "<detailed explanation of the difference>"
    }
  ],
  "findings": [
    {
      "category": "<text|layout|visual|tampering|quality>",
      "description": "<specific finding with exact details>",
      "severity": "<info|warning|critical>"
    }
  ],
  "summary": "<2-3 sentence summary: overall match quality, key differences found, recommendation>"
}

SCORING GUIDELINES:
- 0.95-1.0: Identical documents (only compression/scan artifacts)
- 0.85-0.94: Minor differences (formatting, quality loss)
- 0.70-0.84: Moderate differences (some content variations)
- 0.50-0.69: Significant differences (multiple content changes)
- Below 0.50: Major discrepancies or likely forgery

Be SPECIFIC in differences - include exact text, positions, measurements where possible.`
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
		{Text: "\n\nORIGINAL DOCUMENT (PDF):"},
		{
			InlineData: &GeminiInlineData{
				MimeType: "application/pdf",
				Data:     base64.StdEncoding.EncodeToString(originalPDF),
			},
		},
		{Text: "\n\nPROVIDED DOCUMENTS (Photos for verification):"},
	}

	// Add each photo
	for i, photo := range photos {
		parts = append(parts, GeminiPart{
			Text: fmt.Sprintf("\n\nPhoto %d:", i+1),
		})

		mimeType := detectImageMimeType(photo)
		parts = append(parts, GeminiPart{
			InlineData: &GeminiInlineData{
				MimeType: mimeType,
				Data:     base64.StdEncoding.EncodeToString(photo),
			},
		})
	}

	parts = append(parts, GeminiPart{
		Text: "\n\nAnalyze and compare these documents. Respond with JSON only.",
	})

	return c.sendRequest(ctx, parts)
}

// CompareDocumentsWithPDF compares original PDF with another PDF using Gemini
func (c *GeminiClient) CompareDocumentsWithPDF(ctx context.Context, originalPDF []byte, comparisonPDF []byte) (*entity.DocumentAnalysisResult, *GeminiComparisonResponse, error) {
	parts := []GeminiPart{
		{Text: DocumentComparisonPrompt},
		{Text: "\n\nORIGINAL DOCUMENT (PDF):"},
		{
			InlineData: &GeminiInlineData{
				MimeType: "application/pdf",
				Data:     base64.StdEncoding.EncodeToString(originalPDF),
			},
		},
		{Text: "\n\nPROVIDED DOCUMENT FOR VERIFICATION (PDF):"},
		{
			InlineData: &GeminiInlineData{
				MimeType: "application/pdf",
				Data:     base64.StdEncoding.EncodeToString(comparisonPDF),
			},
		},
		{Text: "\n\nAnalyze and compare these documents. Respond with JSON only."},
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
			Temperature:      0.1, // Low temperature for consistent, deterministic responses
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
	responseText := apiResp.Candidates[0].Content.Parts[0].Text

	var comparisonResp GeminiComparisonResponse
	if err := json.Unmarshal([]byte(responseText), &comparisonResp); err != nil {
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
