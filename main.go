package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ProxyRequest struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	Body      any               `json:"body"`
	Variables []Variable        `json:"variables"`
}

type ProxyResponse struct {
	Status     string            `json:"status"`
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       any               `json:"body"`
	Error      string            `json:"error,omitempty"`
}

type SavedRequest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	Body         any               `json:"body"`
	Params       []QueryParam      `json:"params"`
	Group        string            `json:"group"`
	Description  string            `json:"description"`
	LastResponse *ProxyResponse    `json:"lastResponse,omitempty"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
}

type QueryParam struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type Variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Environment struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Variables []Variable `json:"variables"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
}

type Group struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// parseBodyAsJSON attempts to parse a string body as JSON, returning the parsed object or the original string
func parseBodyAsJSON(body any) any {
	// If it's already not a string, return as-is
	if body == nil {
		return ""
	}

	bodyStr, ok := body.(string)
	if !ok {
		return body // Already parsed or not a string
	}

	// If empty string, return as-is
	if strings.TrimSpace(bodyStr) == "" {
		return bodyStr
	}

	// Try to parse as JSON
	var jsonObj any
	if err := json.Unmarshal([]byte(bodyStr), &jsonObj); err == nil {
		// Successfully parsed as JSON, return the object
		return jsonObj
	}

	// Not valid JSON, return original string
	return bodyStr
}

// bodyToString converts a body (any) to string for HTTP requests
func bodyToString(body any) string {
	if body == nil {
		return ""
	}

	// If it's already a string, return it
	if bodyStr, ok := body.(string); ok {
		return bodyStr
	}

	// If it's a JSON object, marshal it back to string
	if jsonBytes, err := json.Marshal(body); err == nil {
		return string(jsonBytes)
	}

	// Fallback: convert to string representation
	return fmt.Sprintf("%v", body)
}

type SavedRequestsData struct {
	Requests           []SavedRequest `json:"requests"`
	Variables          []Variable     `json:"variables"` // Legacy - kept for backward compatibility
	Environments       []Environment  `json:"environments"`
	CurrentEnvironment string         `json:"currentEnvironment"`
	Groups             []Group        `json:"groups"`
	WordWrap           bool           `json:"wordWrap"`
}

func main() {
	// Create a new chi router
	r := chi.NewRouter()

	// Global middleware
	r.Use(corsMiddleware)
	r.Use(loggingMiddleware)
	r.Use(middleware.Recoverer) // Built-in chi middleware for panic recovery

	// API routes group
	r.Route("/api", func(r chi.Router) {
		r.Post("/proxy", handleProxy)
		r.Get("/health", handleHealth)
		r.Get("/requests", handleRequests)
		r.Post("/requests/save", handleSaveRequest)
		r.Put("/requests/update", handleUpdateRequest)
		r.Delete("/requests/delete", handleDeleteRequest)
		r.Post("/requests/duplicate", handleDuplicateRequest)
		r.Get("/variables", handleVariables)
		r.Post("/variables/save", handleSaveVariables)

		// Environment management endpoints
		r.Get("/environments", handleEnvironments)
		r.Post("/environments", handleCreateEnvironment)
		r.Put("/environments/{id}", handleUpdateEnvironment)
		r.Delete("/environments/{id}", handleDeleteEnvironment)

		// Group management endpoints
		r.Get("/groups", handleGroups)
		r.Post("/groups", handleCreateGroup)
		r.Delete("/groups/{id}", handleDeleteGroup)
		r.Post("/environments/{id}/copy", handleCopyEnvironment)
		r.Post("/environments/{id}/activate", handleActivateEnvironment)

		// UI settings endpoints
		r.Post("/settings/wordwrap", handleSaveWordWrap)
	})

	// Check if frontend/dist exists
	if _, err := os.Stat("frontend/dist"); os.IsNotExist(err) {
		log.Printf("⚠️  Warning: frontend/dist directory not found")
		log.Printf("💡 Run 'cd frontend && npm run build' to build the frontend")
	}

	// Serve static files from frontend/dist directory
	r.Handle("/*", http.FileServer(http.Dir("frontend/dist/")))

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	fmt.Printf("🚀 Postman-like API tester starting on http://localhost:%s\n", port)
	fmt.Println("📁 Serving Svelte frontend from frontend/dist/")
	fmt.Println("🔗 API proxy available at /api/proxy")
	fmt.Println("⏹️  Press Ctrl+C to stop the server")
	fmt.Println("=" + strings.Repeat("=", 50))

	log.Printf("Server listening on port %s", port)

	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Printf("❌ Server failed to start: %v", err)
		fmt.Println("\nPress Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}
}

// corsMiddleware handles CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture status code
		wrapped := &responseWrapper{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("📥 %s %s - %d - %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// handleHealth provides a simple health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "postman-like-api-tester",
	})
}

// handleProxy handles requests to external APIs
func handleProxy(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️  Panic in handleProxy: %v", r)
			respondWithError(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.URL == "" {
		respondWithError(w, "URL is required", http.StatusBadRequest)
		return
	}

	if req.Method == "" {
		req.Method = "GET"
	}

	// Get variables from current environment for template processing
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load environment data: %v", err)
		respondWithError(w, "Failed to load environment data", http.StatusInternalServerError)
		return
	}

	currentEnv, err := getCurrentEnvironment(data)
	if err != nil {
		log.Printf("❌ Failed to get current environment: %v", err)
		respondWithError(w, "Failed to get current environment", http.StatusInternalServerError)
		return
	}

	// Use environment variables instead of request variables for template processing
	req.Variables = currentEnv.Variables

	// Apply template processing to substitute variables
	processedReq := processRequestTemplates(req)
	log.Printf("🔄 Original URL: %s", req.URL)
	if processedReq.URL != req.URL {
		log.Printf("✨ Processed URL: %s", processedReq.URL)
	}

	// Debug headers and template processing
	if len(req.Headers) > 0 {
		log.Printf("📋 Headers: %+v", req.Headers)
		if len(req.Variables) > 0 {
			log.Printf("📋 After template processing: %+v", processedReq.Headers)
		}
	}

	// Make the HTTP request
	response := makeHTTPRequest(processedReq)

	// Return the response to the UI (frontend)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
	}
}

// makeHTTPRequest performs the actual HTTP request to the target API
func makeHTTPRequest(req ProxyRequest) ProxyResponse {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️  Panic in makeHTTPRequest: %v", r)
		}
	}()

	var bodyReader io.Reader
	bodyStr := bodyToString(req.Body)
	if bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		log.Printf("❌ Failed to create request: %v", err)
		return ProxyResponse{
			Error: fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	// Add headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}
	if len(req.Headers) > 0 {
		log.Printf("📋 Set %d headers on HTTP request", len(req.Headers))
	}

	// Make the request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	log.Printf("🔄 Making request to: %s %s", req.Method, req.URL)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("❌ Request failed: %v", err)
		return ProxyResponse{
			Error: fmt.Sprintf("Request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response body: %v", err)
		return ProxyResponse{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("Failed to read response body: %v", err),
		}
	}

	// Convert response headers to map
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0] // Take first value if multiple
		}
	}

	log.Printf("✅ Request completed: %d %s (%d bytes)", resp.StatusCode, resp.Status, len(body))

	// Parse response body as JSON if possible
	responseBody := parseBodyAsJSON(string(body))

	return ProxyResponse{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       responseBody,
	}
}

const requestsFileName = "saved_requests.json"

// Mutex to prevent concurrent file access
var fileAccessMutex sync.RWMutex

// generateID creates a random ID for saved requests
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// processTemplate applies variable substitution to a string using simple find/replace
func processTemplate(input string, variables []Variable) (string, error) {
	if input == "" {
		return input, nil
	}

	result := input
	for _, variable := range variables {
		if variable.Key != "" {
			// Replace {{variableName}} with the variable value
			placeholder := fmt.Sprintf("{{%s}}", variable.Key)
			result = strings.ReplaceAll(result, placeholder, variable.Value)
		}
	}

	return result, nil
}

// processRequestTemplates applies variable substitution to all templated fields in a request
func processRequestTemplates(req ProxyRequest) ProxyRequest {
	// Process URL
	if processedURL, err := processTemplate(req.URL, req.Variables); err == nil {
		req.URL = processedURL
	} else {
		log.Printf("⚠️  Template error in URL: %v", err)
	}

	// Process headers
	processedHeaders := make(map[string]string)
	for key, value := range req.Headers {
		processedKey := key
		processedValue := value

		if newKey, err := processTemplate(key, req.Variables); err == nil {
			processedKey = newKey
		} else {
			log.Printf("⚠️  Template error in header key '%s': %v", key, err)
		}

		if newValue, err := processTemplate(value, req.Variables); err == nil {
			processedValue = newValue
		} else {
			log.Printf("⚠️  Template error in header value '%s': %v", value, err)
		}

		processedHeaders[processedKey] = processedValue
	}
	req.Headers = processedHeaders

	// Process body
	bodyStr := bodyToString(req.Body)
	if processedBodyStr, err := processTemplate(bodyStr, req.Variables); err == nil {
		// Parse the processed body as JSON if possible
		req.Body = parseBodyAsJSON(processedBodyStr)
	} else {
		log.Printf("⚠️  Template error in body: %v", err)
	}

	return req
}

// initializeDefaultEnvironment creates a default environment
func initializeDefaultEnvironment(data *SavedRequestsData) *SavedRequestsData {
	now := time.Now().Format(time.RFC3339)
	defaultEnv := Environment{
		ID:        generateID(),
		Name:      "Default",
		Variables: []Variable{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data.Environments = []Environment{defaultEnv}
	data.CurrentEnvironment = defaultEnv.ID
	return data
}

// migrateRequestsToDefaultGroup migrates requests without groups to default group
func migrateRequestsToDefaultGroup(data *SavedRequestsData) {
	migratedCount := 0
	for i := range data.Requests {
		if data.Requests[i].Group == "" {
			data.Requests[i].Group = "default"
			migratedCount++
		}
	}
	if migratedCount > 0 {
		log.Printf("📦 Migrated %d requests to default group", migratedCount)
	}
}

// migrateStringBodiesToJSON migrates string bodies to parsed JSON objects when possible
func migrateStringBodiesToJSON(data *SavedRequestsData) {
	migratedRequestBodies := 0
	migratedResponseBodies := 0

	for i := range data.Requests {
		// Migrate request body - only migrate if it's currently a string that can be parsed as JSON
		if data.Requests[i].Body != nil {
			if bodyStr, ok := data.Requests[i].Body.(string); ok && strings.TrimSpace(bodyStr) != "" {
				parsedBody := parseBodyAsJSON(bodyStr)
				// Check if parsing resulted in a different type (not a string)
				if _, stillString := parsedBody.(string); !stillString {
					data.Requests[i].Body = parsedBody
					migratedRequestBodies++
				}
			}
		}

		// Migrate response body if it exists - only migrate if it's currently a string that can be parsed as JSON
		if data.Requests[i].LastResponse != nil && data.Requests[i].LastResponse.Body != nil {
			if bodyStr, ok := data.Requests[i].LastResponse.Body.(string); ok && strings.TrimSpace(bodyStr) != "" {
				parsedBody := parseBodyAsJSON(bodyStr)
				// Check if parsing resulted in a different type (not a string)
				if _, stillString := parsedBody.(string); !stillString {
					data.Requests[i].LastResponse.Body = parsedBody
					migratedResponseBodies++
				}
			}
		}
	}

	if migratedRequestBodies > 0 || migratedResponseBodies > 0 {
		log.Printf("🔄 Migrated %d request bodies and %d response bodies from strings to JSON objects",
			migratedRequestBodies, migratedResponseBodies)
	}
}

// migrateVariablesToEnvironments migrates legacy variables to default environment
func migrateVariablesToEnvironments(data *SavedRequestsData) *SavedRequestsData {
	now := time.Now().Format(time.RFC3339)
	defaultEnv := Environment{
		ID:        generateID(),
		Name:      "Default",
		Variables: make([]Variable, len(data.Variables)),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Copy legacy variables to default environment
	copy(defaultEnv.Variables, data.Variables)

	data.Environments = []Environment{defaultEnv}
	data.CurrentEnvironment = defaultEnv.ID

	log.Printf("📦 Migrated %d variables to Default environment", len(data.Variables))
	return data
}

// getCurrentEnvironment returns the current active environment
func getCurrentEnvironment(data *SavedRequestsData) (*Environment, error) {
	if data.CurrentEnvironment == "" && len(data.Environments) > 0 {
		data.CurrentEnvironment = data.Environments[0].ID
	}

	for i := range data.Environments {
		if data.Environments[i].ID == data.CurrentEnvironment {
			return &data.Environments[i], nil
		}
	}

	return nil, fmt.Errorf("current environment not found")
}

// loadSavedRequests reads saved requests from JSON file
func loadSavedRequests() (*SavedRequestsData, error) {
	fileAccessMutex.RLock()
	defer fileAccessMutex.RUnlock()

	data := &SavedRequestsData{
		Requests:     []SavedRequest{},
		Variables:    []Variable{},
		Environments: []Environment{},
	}

	if _, err := os.Stat(requestsFileName); os.IsNotExist(err) {
		// File doesn't exist, create default environment
		data = initializeDefaultEnvironment(data)
		return data, nil
	}

	file, err := os.ReadFile(requestsFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read requests file: %v", err)
	}

	if len(file) == 0 {
		// Empty file, create default environment
		data = initializeDefaultEnvironment(data)
		return data, nil
	}

	if err := json.Unmarshal(file, data); err != nil {
		log.Printf("⚠️  JSON parse error in %s: %v", requestsFileName, err)
		log.Printf("🔧 Attempting to recover by creating new empty file")
		// If JSON is corrupted, create a new file with default environment
		data = initializeDefaultEnvironment(data)
		return data, nil
	}

	// Ensure variables array is not nil (backward compatibility)
	if data.Variables == nil {
		data.Variables = []Variable{}
	}

	// Ensure environments array is not nil
	if data.Environments == nil {
		data.Environments = []Environment{}
	}

	// Migration: If we have legacy variables but no environments, migrate them
	if len(data.Variables) > 0 && len(data.Environments) == 0 {
		data = migrateVariablesToEnvironments(data)
	}

	// Ensure we have at least a default environment
	if len(data.Environments) == 0 {
		data = initializeDefaultEnvironment(data)
	}

	// Ensure current environment is set
	if data.CurrentEnvironment == "" && len(data.Environments) > 0 {
		data.CurrentEnvironment = data.Environments[0].ID
	}

	// Ensure groups array is not nil
	if data.Groups == nil {
		data.Groups = []Group{}
	}

	// Ensure default group exists
	ensureDefaultGroup(data)

	// Migrate existing requests without groups to default group
	migrateRequestsToDefaultGroup(data)

	// Migrate string bodies to parsed JSON objects when possible
	migrateStringBodiesToJSON(data)

	return data, nil
}

// saveSavedRequests writes saved requests to JSON file
func saveSavedRequests(data *SavedRequestsData) error {
	fileAccessMutex.Lock()
	defer fileAccessMutex.Unlock()

	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal requests data: %v", err)
	}

	// On Windows, try direct write first (simpler approach)
	// If that fails, fall back to atomic write with retries
	if err := tryDirectWrite(jsonData); err == nil {
		log.Printf("💾 Saved %d requests to %s", len(data.Requests), requestsFileName)
		return nil
	}

	// Fallback: atomic write with retry logic for Windows file locking issues
	tempFileName := requestsFileName + ".tmp"
	if err := os.WriteFile(tempFileName, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %v", err)
	}

	// Retry rename operation with backoff for Windows file locking
	maxRetries := 5
	baseDelay := 50 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Try to remove target file first (Windows sometimes requires this)
		if _, err := os.Stat(requestsFileName); err == nil {
			os.Remove(requestsFileName)
			time.Sleep(10 * time.Millisecond) // Small delay after removal
		}

		// Attempt rename
		if err := os.Rename(tempFileName, requestsFileName); err == nil {
			log.Printf("💾 Saved %d requests to %s (attempt %d)", len(data.Requests), requestsFileName, attempt)
			return nil
		} else {
			log.Printf("⚠️  Rename attempt %d failed: %v", attempt, err)
			if attempt < maxRetries {
				delay := time.Duration(attempt) * baseDelay
				time.Sleep(delay)
			}
		}
	}

	// If all retries failed, clean up and return error
	os.Remove(tempFileName)
	return fmt.Errorf("failed to save after %d attempts - file may be locked by another process", maxRetries)
}

// tryDirectWrite attempts a direct write to the file (simpler, works most of the time)
func tryDirectWrite(jsonData []byte) error {
	// Try to write directly to the file
	file, err := os.OpenFile(requestsFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		return err
	}

	return file.Sync() // Ensure data is written to disk
}

// handleRequests handles GET requests to retrieve all saved requests
func handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved requests: %v", err)
		respondWithError(w, "Failed to load saved requests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("❌ Failed to encode saved requests: %v", err)
	}
}

// handleSaveRequest handles POST requests to save a new request
func handleSaveRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name         string            `json:"name"`
		URL          string            `json:"url"`
		Method       string            `json:"method"`
		Headers      map[string]string `json:"headers"`
		Body         any               `json:"body"`
		Params       []QueryParam      `json:"params"`
		Group        string            `json:"group"`
		Description  string            `json:"description"`
		LastResponse *ProxyResponse    `json:"lastResponse,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body for save: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondWithError(w, "Request name is required", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		respondWithError(w, "URL is required", http.StatusBadRequest)
		return
	}
	if req.Method == "" {
		req.Method = "GET"
	}

	// Load existing requests
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved requests: %v", err)
		respondWithError(w, "Failed to load saved requests", http.StatusInternalServerError)
		return
	}

	// Ensure default group if none provided
	if req.Group == "" {
		req.Group = "default"
	}

	// Create new saved request
	now := time.Now().Format(time.RFC3339)
	savedReq := SavedRequest{
		ID:           generateID(),
		Name:         req.Name,
		URL:          req.URL,
		Method:       req.Method,
		Headers:      req.Headers,
		Body:         parseBodyAsJSON(req.Body),
		Params:       req.Params,
		Group:        req.Group,
		Description:  req.Description,
		LastResponse: req.LastResponse,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Add to requests list
	data.Requests = append(data.Requests, savedReq)

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save requests: %v", err)
		respondWithError(w, "Failed to save request", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Saved request: %s (%s %s)", savedReq.Name, savedReq.Method, savedReq.URL)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(savedReq); err != nil {
		log.Printf("❌ Failed to encode saved request response: %v", err)
	}
}

// handleUpdateRequest handles PUT requests to update an existing request
func handleUpdateRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID           string            `json:"id"`
		Name         string            `json:"name"`
		URL          string            `json:"url"`
		Method       string            `json:"method"`
		Headers      map[string]string `json:"headers"`
		Body         any               `json:"body"`
		Params       []QueryParam      `json:"params"`
		Group        string            `json:"group"`
		Description  string            `json:"description"`
		LastResponse *ProxyResponse    `json:"lastResponse,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body for update: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ID == "" {
		respondWithError(w, "Request ID is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		respondWithError(w, "Request name is required", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		respondWithError(w, "URL is required", http.StatusBadRequest)
		return
	}
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Group == "" {
		req.Group = "default"
	}

	// Load existing requests
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved requests: %v", err)
		respondWithError(w, "Failed to load saved requests", http.StatusInternalServerError)
		return
	}

	// Find and update the request
	found := false
	for i, existing := range data.Requests {
		if existing.ID == req.ID {
			data.Requests[i].Name = req.Name
			data.Requests[i].URL = req.URL
			data.Requests[i].Method = req.Method
			data.Requests[i].Headers = req.Headers
			data.Requests[i].Body = parseBodyAsJSON(req.Body)
			data.Requests[i].Params = req.Params
			data.Requests[i].Group = req.Group
			data.Requests[i].Description = req.Description
			if req.LastResponse != nil {
				data.Requests[i].LastResponse = req.LastResponse
			}
			data.Requests[i].UpdatedAt = time.Now().Format(time.RFC3339)
			found = true
			break
		}
	}

	if !found {
		respondWithError(w, "Request not found", http.StatusNotFound)
		return
	}

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save updated request: %v", err)
		respondWithError(w, "Failed to save updated request", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Updated request: %s (%s %s)", req.Name, req.Method, req.URL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// handleDeleteRequest handles DELETE requests to delete a request
func handleDeleteRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body for delete: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		respondWithError(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	// Load existing requests
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved requests: %v", err)
		respondWithError(w, "Failed to load saved requests", http.StatusInternalServerError)
		return
	}

	// Find and remove the request
	found := false
	originalCount := len(data.Requests)
	log.Printf("🗑️  Searching for request ID: %s among %d requests", req.ID, originalCount)

	for i, existing := range data.Requests {
		if existing.ID == req.ID {
			log.Printf("🗑️  Found and deleting request: %s (ID: %s)", existing.Name, existing.ID)
			data.Requests = append(data.Requests[:i], data.Requests[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		log.Printf("❌ Request with ID %s not found", req.ID)
		respondWithError(w, "Request not found", http.StatusNotFound)
		return
	}

	newCount := len(data.Requests)
	log.Printf("✅ Request deleted. Count: %d -> %d", originalCount, newCount)

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save after deletion: %v", err)
		respondWithError(w, "Failed to save after deletion", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// handleDuplicateRequest handles POST requests to duplicate a request
func handleDuplicateRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body for duplicate: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		respondWithError(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	// Load existing requests
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved requests: %v", err)
		respondWithError(w, "Failed to load saved requests", http.StatusInternalServerError)
		return
	}

	// Find the request to duplicate
	var originalRequest *SavedRequest
	for _, existing := range data.Requests {
		if existing.ID == req.ID {
			originalRequest = &existing
			break
		}
	}

	if originalRequest == nil {
		respondWithError(w, "Request not found", http.StatusNotFound)
		return
	}

	// Create duplicate
	now := time.Now().Format(time.RFC3339)
	duplicatedReq := SavedRequest{
		ID:           generateID(),
		Name:         originalRequest.Name + " (Copy)",
		URL:          originalRequest.URL,
		Method:       originalRequest.Method,
		Headers:      make(map[string]string),
		Body:         originalRequest.Body,
		Params:       make([]QueryParam, len(originalRequest.Params)),
		Group:        originalRequest.Group,
		Description:  originalRequest.Description,
		LastResponse: nil, // Don't copy response
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Deep copy headers
	for k, v := range originalRequest.Headers {
		duplicatedReq.Headers[k] = v
	}

	// Deep copy params
	copy(duplicatedReq.Params, originalRequest.Params)

	// Add to requests list
	data.Requests = append(data.Requests, duplicatedReq)

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save duplicated request: %v", err)
		respondWithError(w, "Failed to save duplicated request", http.StatusInternalServerError)
		return
	}

	log.Printf("📋 Duplicated request: %s -> %s", originalRequest.Name, duplicatedReq.Name)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(duplicatedReq); err != nil {
		log.Printf("❌ Failed to encode duplicated request response: %v", err)
	}
}

// handleVariables handles GET requests to retrieve variables from current environment
func handleVariables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load variables: %v", err)
		respondWithError(w, "Failed to load variables", http.StatusInternalServerError)
		return
	}

	// Get current environment
	currentEnv, err := getCurrentEnvironment(data)
	if err != nil {
		log.Printf("❌ Failed to get current environment: %v", err)
		respondWithError(w, "Failed to get current environment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string][]Variable{"variables": currentEnv.Variables}); err != nil {
		log.Printf("❌ Failed to encode variables: %v", err)
	}
}

// handleSaveVariables handles POST requests to save variables to current environment
func handleSaveVariables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Variables []Variable `json:"variables"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body for save variables: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Load existing data
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved data: %v", err)
		respondWithError(w, "Failed to load saved data", http.StatusInternalServerError)
		return
	}

	// Find and update current environment
	found := false
	for i := range data.Environments {
		if data.Environments[i].ID == data.CurrentEnvironment {
			data.Environments[i].Variables = req.Variables
			data.Environments[i].UpdatedAt = time.Now().Format(time.RFC3339)
			found = true
			break
		}
	}

	if !found {
		log.Printf("❌ Current environment not found: %s", data.CurrentEnvironment)
		respondWithError(w, "Current environment not found", http.StatusInternalServerError)
		return
	}

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save variables: %v", err)
		respondWithError(w, "Failed to save variables", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Saved %d variables to environment %s", len(req.Variables), data.CurrentEnvironment)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "saved"}); err != nil {
		log.Printf("❌ Failed to encode variables response: %v", err)
	}
}

// handleEnvironments handles GET requests to list all environments
func handleEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load environments: %v", err)
		respondWithError(w, "Failed to load environments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"environments":       data.Environments,
		"currentEnvironment": data.CurrentEnvironment,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode environments: %v", err)
	}
}

// handleCreateEnvironment handles POST requests to create a new environment
func handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body for create environment: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondWithError(w, "Environment name is required", http.StatusBadRequest)
		return
	}

	// Load existing data
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved data: %v", err)
		respondWithError(w, "Failed to load saved data", http.StatusInternalServerError)
		return
	}

	// Check if environment name already exists
	for _, env := range data.Environments {
		if env.Name == req.Name {
			respondWithError(w, "Environment name already exists", http.StatusConflict)
			return
		}
	}

	// Create new environment
	now := time.Now().Format(time.RFC3339)
	newEnv := Environment{
		ID:        generateID(),
		Name:      req.Name,
		Variables: []Variable{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data.Environments = append(data.Environments, newEnv)

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save environment: %v", err)
		respondWithError(w, "Failed to save environment", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Created environment: %s (%s)", newEnv.Name, newEnv.ID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(newEnv); err != nil {
		log.Printf("❌ Failed to encode environment response: %v", err)
	}
}

// handleUpdateEnvironment handles PUT requests to update an environment
func handleUpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	envID := chi.URLParam(r, "id")
	if envID == "" {
		respondWithError(w, "Environment ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Name      string     `json:"name"`
		Variables []Variable `json:"variables"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body for update environment: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Load existing data
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved data: %v", err)
		respondWithError(w, "Failed to load saved data", http.StatusInternalServerError)
		return
	}

	// Find and update environment
	found := false
	for i := range data.Environments {
		if data.Environments[i].ID == envID {
			if req.Name != "" {
				// Check if new name conflicts with existing environments
				for j, env := range data.Environments {
					if j != i && env.Name == req.Name {
						respondWithError(w, "Environment name already exists", http.StatusConflict)
						return
					}
				}
				data.Environments[i].Name = req.Name
			}
			if req.Variables != nil {
				data.Environments[i].Variables = req.Variables
			}
			data.Environments[i].UpdatedAt = time.Now().Format(time.RFC3339)
			found = true
			break
		}
	}

	if !found {
		respondWithError(w, "Environment not found", http.StatusNotFound)
		return
	}

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save environment: %v", err)
		respondWithError(w, "Failed to save environment", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Updated environment: %s", envID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "updated"}); err != nil {
		log.Printf("❌ Failed to encode environment response: %v", err)
	}
}

// handleDeleteEnvironment handles DELETE requests to delete an environment
func handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	envID := chi.URLParam(r, "id")
	if envID == "" {
		respondWithError(w, "Environment ID is required", http.StatusBadRequest)
		return
	}

	// Load existing data
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved data: %v", err)
		respondWithError(w, "Failed to load saved data", http.StatusInternalServerError)
		return
	}

	// Don't allow deleting the last environment
	if len(data.Environments) <= 1 {
		respondWithError(w, "Cannot delete the last environment", http.StatusBadRequest)
		return
	}

	// Find and remove environment
	found := false
	newEnvironments := []Environment{}
	for _, env := range data.Environments {
		if env.ID != envID {
			newEnvironments = append(newEnvironments, env)
		} else {
			found = true
		}
	}

	if !found {
		respondWithError(w, "Environment not found", http.StatusNotFound)
		return
	}

	data.Environments = newEnvironments

	// If we deleted the current environment, switch to the first available
	if data.CurrentEnvironment == envID {
		data.CurrentEnvironment = data.Environments[0].ID
	}

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save environments: %v", err)
		respondWithError(w, "Failed to save environments", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Deleted environment: %s", envID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted"}); err != nil {
		log.Printf("❌ Failed to encode environment response: %v", err)
	}
}

// handleCopyEnvironment handles POST requests to copy variables between environments
func handleCopyEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetEnvID := chi.URLParam(r, "id")
	if targetEnvID == "" {
		respondWithError(w, "Target environment ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		SourceEnvironmentID string `json:"sourceEnvironmentId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body for copy environment: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SourceEnvironmentID == "" {
		respondWithError(w, "Source environment ID is required", http.StatusBadRequest)
		return
	}

	// Load existing data
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved data: %v", err)
		respondWithError(w, "Failed to load saved data", http.StatusInternalServerError)
		return
	}

	// Find source environment
	var sourceEnv *Environment
	for _, env := range data.Environments {
		if env.ID == req.SourceEnvironmentID {
			sourceEnv = &env
			break
		}
	}

	if sourceEnv == nil {
		respondWithError(w, "Source environment not found", http.StatusNotFound)
		return
	}

	// Find and update target environment
	found := false
	for i := range data.Environments {
		if data.Environments[i].ID == targetEnvID {
			// Copy variables from source to target
			data.Environments[i].Variables = make([]Variable, len(sourceEnv.Variables))
			copy(data.Environments[i].Variables, sourceEnv.Variables)
			data.Environments[i].UpdatedAt = time.Now().Format(time.RFC3339)
			found = true
			break
		}
	}

	if !found {
		respondWithError(w, "Target environment not found", http.StatusNotFound)
		return
	}

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save environment: %v", err)
		respondWithError(w, "Failed to save environment", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Copied %d variables from %s to %s", len(sourceEnv.Variables), req.SourceEnvironmentID, targetEnvID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "copied"}); err != nil {
		log.Printf("❌ Failed to encode copy response: %v", err)
	}
}

// handleActivateEnvironment handles POST requests to activate an environment
func handleActivateEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	envID := chi.URLParam(r, "id")
	if envID == "" {
		respondWithError(w, "Environment ID is required", http.StatusBadRequest)
		return
	}

	// Load existing data
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved data: %v", err)
		respondWithError(w, "Failed to load saved data", http.StatusInternalServerError)
		return
	}

	// Check if environment exists
	found := false
	for _, env := range data.Environments {
		if env.ID == envID {
			found = true
			break
		}
	}

	if !found {
		respondWithError(w, "Environment not found", http.StatusNotFound)
		return
	}

	// Set as current environment
	data.CurrentEnvironment = envID

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save current environment: %v", err)
		respondWithError(w, "Failed to save current environment", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Activated environment: %s", envID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "activated"}); err != nil {
		log.Printf("❌ Failed to encode activation response: %v", err)
	}
}

// handleGroups handles GET requests to get all groups
func handleGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved requests: %v", err)
		respondWithError(w, "Failed to load saved requests", http.StatusInternalServerError)
		return
	}

	// Ensure default group exists
	ensureDefaultGroup(data)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string][]Group{"groups": data.Groups}); err != nil {
		log.Printf("❌ Failed to encode groups: %v", err)
	}
}

// handleCreateGroup handles POST requests to create a new group
func handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body for create group: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondWithError(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Load existing data
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved requests: %v", err)
		respondWithError(w, "Failed to load saved requests", http.StatusInternalServerError)
		return
	}

	// Check if group already exists
	for _, group := range data.Groups {
		if group.Name == req.Name {
			respondWithError(w, "Group already exists", http.StatusConflict)
			return
		}
	}

	// Create new group
	now := time.Now().Format(time.RFC3339)
	newGroup := Group{
		ID:        generateID(),
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	data.Groups = append(data.Groups, newGroup)

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save group: %v", err)
		respondWithError(w, "Failed to save group", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Created group: %s", newGroup.Name)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(newGroup); err != nil {
		log.Printf("❌ Failed to encode group response: %v", err)
	}
}

// handleDeleteGroup handles DELETE requests to delete a group
func handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID := chi.URLParam(r, "id")
	if groupID == "" {
		respondWithError(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Load existing data
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load saved requests: %v", err)
		respondWithError(w, "Failed to load saved requests", http.StatusInternalServerError)
		return
	}

	// Find the group and check if it has requests
	var groupName string
	found := false
	for _, group := range data.Groups {
		if group.ID == groupID {
			groupName = group.Name
			found = true
			break
		}
	}

	if !found {
		respondWithError(w, "Group not found", http.StatusNotFound)
		return
	}

	// Don't allow deleting default group
	if groupName == "default" {
		respondWithError(w, "Cannot delete default group", http.StatusBadRequest)
		return
	}

	// Check if group has any requests
	hasRequests := false
	for _, req := range data.Requests {
		if req.Group == groupName {
			hasRequests = true
			break
		}
	}

	if hasRequests {
		respondWithError(w, "Cannot delete group with requests", http.StatusBadRequest)
		return
	}

	// Remove the group
	for i, group := range data.Groups {
		if group.ID == groupID {
			data.Groups = append(data.Groups[:i], data.Groups[i+1:]...)
			break
		}
	}

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save after group deletion: %v", err)
		respondWithError(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Deleted group: %s", groupName)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted"}); err != nil {
		log.Printf("❌ Failed to encode delete response: %v", err)
	}
}

// handleSaveWordWrap saves the word wrap setting
func handleSaveWordWrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WordWrap bool `json:"wordWrap"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid word wrap request body: %v", err)
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Load current data
	data, err := loadSavedRequests()
	if err != nil {
		log.Printf("❌ Failed to load data for word wrap update: %v", err)
		respondWithError(w, "Failed to load data", http.StatusInternalServerError)
		return
	}

	// Update word wrap setting
	data.WordWrap = req.WordWrap

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save word wrap setting: %v", err)
		respondWithError(w, "Failed to save word wrap setting", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Updated word wrap setting to: %t", req.WordWrap)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"wordWrap": req.WordWrap}); err != nil {
		log.Printf("❌ Failed to encode word wrap response: %v", err)
	}
}

// ensureDefaultGroup ensures the default group exists
func ensureDefaultGroup(data *SavedRequestsData) {
	// Check if default group exists
	for _, group := range data.Groups {
		if group.Name == "default" {
			return
		}
	}

	// Create default group
	now := time.Now().Format(time.RFC3339)
	defaultGroup := Group{
		ID:        generateID(),
		Name:      "default",
		CreatedAt: now,
		UpdatedAt: now,
	}

	data.Groups = append(data.Groups, defaultGroup)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ProxyResponse{
		Error: message,
	})
}
