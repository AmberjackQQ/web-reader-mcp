package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// MCP JSON-RPC message structures
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// MCP Protocol structures
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      map[string]string      `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      map[string]string      `json:"serverInfo"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	_         string                 `json:"_meta,omitempty"` // Optional metadata
}

// Tool input structures
type WebReaderInput struct {
	URL               string  `json:"url"`
	Model             string  `json:"model,omitempty"`
	MaxTokens         int     `json:"maxTokens,omitempty"`
	Temperature       float64 `json:"temperature,omitempty"`
	RetainImages      bool    `json:"retain_images,omitempty"`
	KeepImageDataURL  bool    `json:"keep_img_data_url,omitempty"`
	WithImagesSummary bool    `json:"with_images_summary,omitempty"`
	WithLinksSummary  bool    `json:"with_links_summary,omitempty"`
}

// AI API structures
type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIRequest struct {
	Temperature      float64     `json:"temperature"`
	TopK            int         `json:"top_k"`
	TopP            float64     `json:"top_p"`
	FrequencyPenalty float64     `json:"frequency_penalty"`
	Messages        []AIMessage `json:"messages"`
	Model           string      `json:"model"`
	MaxTokens       int         `json:"maxTokens"`
}

type AIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Content structures for tool responses
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ImageContent struct {
	Type string `json:"type"`
	Data string `json:"data"`
	MIME string `json:"mime_type,omitempty"`
}

// Image and link metadata structures
type ImageInfo struct {
	OriginalURL string `json:"original_url"`
	DataURL     string `json:"data_url,omitempty"`
	Alt         string `json:"alt,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Size        int64  `json:"size_bytes,omitempty"`
}

type LinkInfo struct {
	URL   string `json:"url"`
	Text  string `json:"text,omitempty"`
	Title string `json:"title,omitempty"`
}

const (
	aiAPIURL      = "https://api.gitcode.com/api/v5/chat/completions"
	defaultModel  = "deepseek-ai/DeepSeek-V3"
	defaultMaxTokens = 4000
	maxImageSize  = 5 * 1024 * 1024
	mcpVersion    = "2024-11-05"
)

var (
	apiKey string
)

func main() {
	// Get API key from environment
	apiKey = os.Getenv("AI_API_KEY")
	if apiKey == "" {
		log.Fatal("AI_API_KEY environment variable is not set")
	}

	log.SetPrefix("[Web-Reader MCP] ")
	log.Println("Starting Web Reader MCP Server (stdio mode)...")

	// Start processing stdin/stdout
	processStdio()
}

// processStdio handles JSON-RPC communication via stdin/stdout
func processStdio() {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var message JSONRPCMessage
		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				log.Println("Received EOF, shutting down...")
				return
			}
			log.Printf("Error decoding message: %v", err)
			continue
		}

		log.Printf("Received message: method=%s, id=%v", message.Method, message.ID)

		response := handleMessage(&message)

		if err := encoder.Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	}
}

// handleMessage dispatches incoming RPC messages to appropriate handlers
func handleMessage(msg *JSONRPCMessage) *JSONRPCMessage {
	switch msg.Method {
	case "initialize":
		return handleInitialize(msg)
	case "tools/list":
		return handleListTools(msg)
	case "tools/call":
		return handleCallTool(msg)
	case "ping":
		return handlePing(msg)
	default:
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", msg.Method),
			},
		}
	}
}

// handleInitialize responds to the initialize request
func handleInitialize(msg *JSONRPCMessage) *JSONRPCMessage {
	var params InitializeParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		log.Printf("Error unmarshaling initialize params: %v", err)
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	log.Printf("Initialize request from client: %s", params.ClientInfo["name"])

	result := InitializeResult{
		ProtocolVersion: mcpVersion,
		Capabilities: map[string]interface{}{
			"tools": map[string]bool{},
			"resources": map[string]bool{},
		},
		ServerInfo: map[string]string{
			"name":    "web-reader-mcp",
			"version": "2.0.0",
		},
	}

	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  result,
	}
}

// handleListTools returns the list of available tools
func handleListTools(msg *JSONRPCMessage) *JSONRPCMessage {
	tools := []Tool{
		{
			Name:        "web_reader",
			Description: "Fetch web content and convert it to clean Markdown format. Optionally extract images and links with metadata.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to fetch content from",
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "AI model to use for conversion (default: deepseek-chat)",
					},
					"maxTokens": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum tokens in response (default: 4000)",
					},
					"temperature": map[string]interface{}{
						"type":        "number",
						"description": "AI temperature 0-1 (default: 0.7)",
					},
					"retain_images": map[string]interface{}{
						"type":        "boolean",
						"description": "Extract images from content",
					},
					"keep_img_data_url": map[string]interface{}{
						"type":        "boolean",
						"description": "Download and convert images to base64 data URLs",
					},
					"with_images_summary": map[string]interface{}{
						"type":        "boolean",
						"description": "Include image metadata in response",
					},
					"with_links_summary": map[string]interface{}{
						"type":        "boolean",
						"description": "Extract and include link metadata",
					},
				},
				"required": []string{"url"},
			},
		},
	}

	result := ListToolsResult{
		Tools: tools,
	}

	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  result,
	}
}

// handleCallTool executes a tool call
func handleCallTool(msg *JSONRPCMessage) *JSONRPCMessage {
	var params CallToolParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	log.Printf("Tool call: %s", params.Name)

	switch params.Name {
	case "web_reader":
		return handleWebReader(msg.ID, params.Arguments)
	default:
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Unknown tool: %s", params.Name),
			},
		}
	}
}

// handleWebReader processes the web_reader tool call
func handleWebReader(id interface{}, args map[string]interface{}) *JSONRPCMessage {
	startTime := time.Now()

	// Parse input arguments
	input, err := parseWebReaderInput(args)
	if err != nil {
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -32602,
				Message: err.Error(),
			},
		}
	}

	log.Printf("Fetching URL: %s", input.URL)

	// Step 1: Fetch web content
	htmlContent, err := fetchWebContent(input.URL)
	if err != nil {
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -1,
				Message: fmt.Sprintf("Failed to fetch web content: %v", err),
			},
		}
	}

	// Step 2: Extract images and links if requested
	var images []ImageInfo
	var links []LinkInfo

	parsedURL, _ := url.Parse(input.URL)

	if input.RetainImages || input.WithImagesSummary {
		log.Println("Extracting images...")
		images, _ = extractImages(htmlContent, parsedURL, input.KeepImageDataURL)
	}

	if input.WithLinksSummary {
		log.Println("Extracting links...")
		links = extractLinks(htmlContent, parsedURL)
	}

	// Step 3: Convert to Markdown using AI
	log.Println("Converting to Markdown...")
	markdownContent, err := convertToMarkdown(htmlContent, input.Model, input.MaxTokens, input.Temperature)
	if err != nil {
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -2,
				Message: fmt.Sprintf("Failed to convert content: %v", err),
			},
		}
	}

	// Step 4: Post-process Markdown if needed
	if input.RetainImages && input.KeepImageDataURL && len(images) > 0 {
		markdownContent = updateImageReferences(markdownContent, images)
	}

	// Step 5: Build response content
	processingTime := float64(time.Since(startTime).Microseconds()) / 1000.0
	content := buildToolResponse(markdownContent, input.URL, processingTime, images, links)

	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": content,
		},
	}
}

// parseWebReaderInput parses and validates the tool input arguments
func parseWebReaderInput(args map[string]interface{}) (*WebReaderInput, error) {
	input := &WebReaderInput{}

	// Required parameter: url
	if urlVal, ok := args["url"].(string); ok {
		input.URL = urlVal
	} else {
		return nil, fmt.Errorf("missing required parameter: url")
	}

	// Validate URL format
	if _, err := url.Parse(input.URL); err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	// Optional parameters
	if v, ok := args["model"].(string); ok {
		input.Model = v
	}
	if v, ok := args["maxTokens"].(float64); ok {
		input.MaxTokens = int(v)
	}
	if v, ok := args["temperature"].(float64); ok {
		input.Temperature = v
	}
	if v, ok := args["retain_images"].(bool); ok {
		input.RetainImages = v
	}
	if v, ok := args["keep_img_data_url"].(bool); ok {
		input.KeepImageDataURL = v
	}
	if v, ok := args["with_images_summary"].(bool); ok {
		input.WithImagesSummary = v
	}
	if v, ok := args["with_links_summary"].(bool); ok {
		input.WithLinksSummary = v
	}

	return input, nil
}

// buildToolResponse constructs the content array for the tool response
func buildToolResponse(markdown, sourceURL string, processingTime float64, images []ImageInfo, links []LinkInfo) []interface{} {
	content := []interface{}{
		TextContent{
			Type: "text",
			Text: fmt.Sprintf("# Web Content from %s\n\n", sourceURL),
		},
		TextContent{
			Type: "text",
			Text: markdown,
		},
	}

	// Add metadata summary
	metadata := fmt.Sprintf("\n\n---\n**Metadata:**\n")
	metadata += fmt.Sprintf("- Source: %s\n", sourceURL)
	metadata += fmt.Sprintf("- Processing time: %.2fms\n", processingTime)
	metadata += fmt.Sprintf("- Word count: %d\n", len(strings.Fields(markdown)))
	metadata += fmt.Sprintf("- Images found: %d\n", len(images))
	metadata += fmt.Sprintf("- Links found: %d\n", len(links))

	if len(images) > 0 {
		metadata += "\n**Images:**\n"
		for i, img := range images {
			if i >= 10 { // Limit to first 10
				metadata += fmt.Sprintf("... and %d more\n", len(images)-10)
				break
			}
			metadata += fmt.Sprintf("%d. %s", i+1, img.OriginalURL)
			if img.Alt != "" {
				metadata += fmt.Sprintf(" (Alt: %s)", img.Alt)
			}
			if img.Width > 0 && img.Height > 0 {
				metadata += fmt.Sprintf(" [%dx%d]", img.Width, img.Height)
			}
			metadata += "\n"
		}
	}

	if len(links) > 0 {
		metadata += "\n**Links:**\n"
		for i, link := range links {
			if i >= 10 { // Limit to first 10
				metadata += fmt.Sprintf("... and %d more\n", len(links)-10)
				break
			}
			metadata += fmt.Sprintf("%d. %s", i+1, link.URL)
			if link.Text != "" {
				metadata += fmt.Sprintf(" - %s", truncateString(link.Text, 50))
			}
			metadata += "\n"
		}
	}

	content = append(content, TextContent{
		Type: "text",
		Text: metadata,
	})

	return content
}

// handlePing responds to ping requests
func handlePing(msg *JSONRPCMessage) *JSONRPCMessage {
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  "pong",
	}
}

// fetchWebContent fetches the HTML content from the given URL
func fetchWebContent(targetURL string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// extractImages extracts all images from HTML content
func extractImages(htmlContent string, baseURL *url.URL, keepDataURL bool) ([]ImageInfo, error) {
	var images []ImageInfo

	imgRegex := regexp.MustCompile(`<img[^>]+>`)
	matches := imgRegex.FindAllString(htmlContent, -1)

	srcRegex := regexp.MustCompile(`src=["']([^"']+)["']`)
	altRegex := regexp.MustCompile(`alt=["']([^"']*)["']`)
	widthRegex := regexp.MustCompile(`width=["'](\d+)["']`)
	heightRegex := regexp.MustCompile(`height=["'](\d+)["']`)

	for _, imgTag := range matches {
		srcMatch := srcRegex.FindStringSubmatch(imgTag)
		if len(srcMatch) < 2 {
			continue
		}

		imgSrc := srcMatch[1]
		if strings.HasPrefix(imgSrc, "data:") || imgSrc == "" {
			continue
		}

		imgURL, err := resolveURL(baseURL, imgSrc)
		if err != nil {
			continue
		}

		imageInfo := ImageInfo{
			OriginalURL: imgURL.String(),
		}

		if altMatch := altRegex.FindStringSubmatch(imgTag); len(altMatch) >= 2 {
			imageInfo.Alt = altMatch[1]
		}

		if widthMatch := widthRegex.FindStringSubmatch(imgTag); len(widthMatch) >= 2 {
			fmt.Sscanf(widthMatch[1], "%d", &imageInfo.Width)
		}
		if heightMatch := heightRegex.FindStringSubmatch(imgTag); len(heightMatch) >= 2 {
			fmt.Sscanf(heightMatch[1], "%d", &imageInfo.Height)
		}

		if keepDataURL {
			if dataURL, size, err := downloadAndConvertImage(imgURL.String()); err == nil {
				imageInfo.DataURL = dataURL
				imageInfo.Size = size
			}
		}

		images = append(images, imageInfo)
	}

	return images, nil
}

// downloadAndConvertImage downloads an image and converts it to base64 data URL
func downloadAndConvertImage(imgURL string) (string, int64, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Get(imgURL)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if resp.ContentLength > maxImageSize {
		return "", 0, fmt.Errorf("image too large: %d bytes", resp.ContentLength)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize))
	if err != nil {
		return "", 0, err
	}

	if len(data) > maxImageSize {
		return "", 0, fmt.Errorf("image too large: %d bytes", len(data))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data))

	return dataURL, int64(len(data)), nil
}

// extractLinks extracts all links from HTML content
func extractLinks(htmlContent string, baseURL *url.URL) []LinkInfo {
	var links []LinkInfo

	linkRegex := regexp.MustCompile(`<a[^>]+>(.*?)</a>`)
	matches := linkRegex.FindAllStringSubmatch(htmlContent, -1)

	hrefRegex := regexp.MustCompile(`href=["']([^"']+)["']`)

	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		linkTag := match[0]
		linkText := strings.TrimSpace(match[1])

		textRegex := regexp.MustCompile(`<[^>]+>`)
		linkText = textRegex.ReplaceAllString(linkText, "")
		linkText = strings.TrimSpace(linkText)

		hrefMatch := hrefRegex.FindStringSubmatch(linkTag)
		if len(hrefMatch) < 2 {
			continue
		}

		href := hrefMatch[1]

		if strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") ||
			strings.HasPrefix(href, "#") {
			continue
		}

		linkURL, err := resolveURL(baseURL, href)
		if err != nil {
			continue
		}

		linkStr := linkURL.String()

		if seen[linkStr] {
			continue
		}
		seen[linkStr] = true

		linkInfo := LinkInfo{
			URL:   linkStr,
			Text:  linkText,
			Title: extractTitle(linkTag),
		}

		links = append(links, linkInfo)
	}

	return links
}

// extractTitle extracts the title attribute from a link tag
func extractTitle(linkTag string) string {
	titleRegex := regexp.MustCompile(`title=["']([^"']*)["']`)
	if match := titleRegex.FindStringSubmatch(linkTag); len(match) >= 2 {
		return match[1]
	}
	return ""
}

// resolveURL resolves a potentially relative URL against a base URL
func resolveURL(base *url.URL, ref string) (*url.URL, error) {
	refURL, err := url.Parse(ref)
	if err != nil {
		return nil, err
	}
	return base.ResolveReference(refURL), nil
}

// updateImageReferences updates image references in Markdown with data URLs
func updateImageReferences(markdown string, images []ImageInfo) string {
	for _, img := range images {
		if img.DataURL != "" {
			markdown = strings.ReplaceAll(markdown, img.OriginalURL, img.DataURL)
			strippedURL := stripURLFragment(img.OriginalURL)
			markdown = strings.ReplaceAll(markdown, strippedURL, img.DataURL)
		}
	}
	return markdown
}

// stripURLFragment removes query string and fragment from URL
func stripURLFragment(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// convertToMarkdown calls the AI API to convert HTML to Markdown
func convertToMarkdown(htmlContent, model string, maxTokens int, temperature float64) (string, error) {
	if model == "" {
		model = defaultModel
	}
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}
	if temperature == 0 {
		temperature = 0.7
	}

	messages := []AIMessage{
		{
			Role: "system",
			Content: "You are a web content extractor. Your task is to convert HTML content to clean, well-formatted Markdown. " +
				"Extract only the main content, removing ads, navigation, scripts, and other non-essential elements. " +
				"Preserve the structure with proper Markdown headings, lists, links, and formatting. " +
				"Keep all image references in Markdown format: ![alt text](image_url). " +
				"Keep all links in Markdown format: [link text](url). " +
				"Return ONLY the Markdown content without any explanations or additional text.",
		},
		{
			Role: "user",
			Content: "Please convert the following HTML content to Markdown:\n\n" + htmlContent,
		},
	}

	aiReq := AIRequest{
		Temperature:       temperature,
		TopK:              0,
		TopP:              0,
		FrequencyPenalty:  0,
		Messages:          messages,
		Model:             model,
		MaxTokens:         maxTokens,
	}

	payload, err := json.Marshal(aiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequest("POST", aiAPIURL, strings.NewReader(string(payload)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call AI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var aiResp AIResponse
	if err := json.Unmarshal(body, &aiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if aiResp.Error != nil {
		return "", fmt.Errorf("AI API error: %s", aiResp.Error.Message)
	}

	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI API")
	}

	content := strings.TrimSpace(aiResp.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("empty response from AI API")
	}

	return content, nil
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
