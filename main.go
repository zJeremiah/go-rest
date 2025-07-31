package main

import (
	"bytes"
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
	"text/template"
	"time"
)

type ProxyRequest struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	Variables []Variable        `json:"variables"`
}

type ProxyResponse struct {
	Status     string            `json:"status"`
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
}

type SavedRequest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	Params       []QueryParam      `json:"params"`
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

type SavedRequestsData struct {
	Requests  []SavedRequest `json:"requests"`
	Variables []Variable     `json:"variables"`
}

func main() {

	// Create a new ServeMux
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/proxy", enableCORS(handleProxy))
	mux.HandleFunc("/api/health", enableCORS(handleHealth))
	mux.HandleFunc("/api/requests", enableCORS(handleRequests))
	mux.HandleFunc("/api/requests/save", enableCORS(handleSaveRequest))
	mux.HandleFunc("/api/requests/update", enableCORS(handleUpdateRequest))
	mux.HandleFunc("/api/requests/delete", enableCORS(handleDeleteRequest))
	mux.HandleFunc("/api/requests/duplicate", enableCORS(handleDuplicateRequest))
	mux.HandleFunc("/api/variables", enableCORS(handleVariables))
	mux.HandleFunc("/api/variables/save", enableCORS(handleSaveVariables))

	// Check if frontend/dist exists
	if _, err := os.Stat("frontend/dist"); os.IsNotExist(err) {
		log.Printf("⚠️  Warning: frontend/dist directory not found")
		log.Printf("💡 Run 'cd frontend && npm run build' to build the frontend")
	}

	// Serve static files from frontend/dist directory
	mux.Handle("/", http.FileServer(http.Dir("frontend/dist/")))

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

	// Wrap mux with logging middleware
	handler := loggingMiddleware(mux)

	err := http.ListenAndServe(":"+port, handler)
	if err != nil {
		log.Printf("❌ Server failed to start: %v", err)
		fmt.Println("\nPress Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}
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

// enableCORS wraps handlers with CORS headers
func enableCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
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

	// Return the response
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
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
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

	// Read response body with size limit
	const maxBodySize = 10 * 1024 * 1024 // 10MB limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
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

	return ProxyResponse{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(body),
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

// processTemplate applies variable substitution to a string using Go templates
func processTemplate(input string, variables []Variable) (string, error) {
	if input == "" {
		return input, nil
	}

	// Create template data map
	data := make(map[string]string)
	for _, variable := range variables {
		if variable.Key != "" {
			data[variable.Key] = variable.Value
		}
	}

	// Parse and execute template
	tmpl, err := template.New("template").Parse(input)
	if err != nil {
		return input, fmt.Errorf("template parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return input, fmt.Errorf("template execute error: %v", err)
	}

	return buf.String(), nil
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
	if processedBody, err := processTemplate(req.Body, req.Variables); err == nil {
		req.Body = processedBody
	} else {
		log.Printf("⚠️  Template error in body: %v", err)
	}

	return req
}

// loadSavedRequests reads saved requests from JSON file
func loadSavedRequests() (*SavedRequestsData, error) {
	fileAccessMutex.RLock()
	defer fileAccessMutex.RUnlock()

	data := &SavedRequestsData{Requests: []SavedRequest{}, Variables: []Variable{}}

	if _, err := os.Stat(requestsFileName); os.IsNotExist(err) {
		// File doesn't exist, return empty data
		return data, nil
	}

	file, err := os.ReadFile(requestsFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read requests file: %v", err)
	}

	if len(file) == 0 {
		// Empty file, return empty data
		return data, nil
	}

	if err := json.Unmarshal(file, data); err != nil {
		log.Printf("⚠️  JSON parse error in %s: %v", requestsFileName, err)
		log.Printf("🔧 Attempting to recover by creating new empty file")
		// If JSON is corrupted, create a new empty file
		return data, nil
	}

	// Ensure variables array is not nil
	if data.Variables == nil {
		data.Variables = []Variable{}
	}

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
		Body         string            `json:"body"`
		Params       []QueryParam      `json:"params"`
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

	// Create new saved request
	now := time.Now().Format(time.RFC3339)
	savedReq := SavedRequest{
		ID:           generateID(),
		Name:         req.Name,
		URL:          req.URL,
		Method:       req.Method,
		Headers:      req.Headers,
		Body:         req.Body,
		Params:       req.Params,
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
		Body         string            `json:"body"`
		Params       []QueryParam      `json:"params"`
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
			data.Requests[i].Body = req.Body
			data.Requests[i].Params = req.Params
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
		Headers:      originalRequest.Headers,
		Body:         originalRequest.Body,
		Params:       originalRequest.Params,
		LastResponse: nil, // Don't copy the response
		CreatedAt:    now,
		UpdatedAt:    now,
	}

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

// handleVariables handles GET requests to retrieve all variables
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string][]Variable{"variables": data.Variables}); err != nil {
		log.Printf("❌ Failed to encode variables: %v", err)
	}
}

// handleSaveVariables handles POST requests to save variables
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

	// Update variables
	data.Variables = req.Variables

	// Save to file
	if err := saveSavedRequests(data); err != nil {
		log.Printf("❌ Failed to save variables: %v", err)
		respondWithError(w, "Failed to save variables", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Saved %d variables", len(req.Variables))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "saved"}); err != nil {
		log.Printf("❌ Failed to encode variables response: %v", err)
	}
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ProxyResponse{
		Error: message,
	})
}
