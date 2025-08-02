<script>
  import { createEventDispatcher } from 'svelte';
  
  export let response = null;
  export let loading = false;
  export let wordWrap = false;

  const dispatch = createEventDispatcher();
  let hljs;
  let responseBodyElement;
  let highlightLoaded = false;
  let activeTab = 'body';

  const tabs = [
    { id: 'body', label: 'Response Body', icon: '📄' },
    { id: 'headers', label: 'Headers', icon: '📋' }
  ];

  async function loadHighlight() {
    if (!highlightLoaded && typeof window !== 'undefined') {
      try {
        hljs = await import('highlight.js');
        await import('highlight.js/styles/github.css');
        highlightLoaded = true;
      } catch (error) {
        console.warn('Failed to load syntax highlighting:', error);
      }
    }
  }

  function formatJSON(data) {
    try {
      // If it's already an object, stringify it directly
      if (typeof data === 'object' && data !== null) {
        return JSON.stringify(data, null, 2);
      }
      
      // If it's a string, parse then stringify
      if (typeof data === 'string') {
        return JSON.stringify(JSON.parse(data), null, 2);
      }
      
      // For other types, convert to string
      return String(data);
    } catch {
      return String(data);
    }
  }

  function getStatusClass(statusCode) {
    if (!statusCode) return 'status-error';
    if (statusCode >= 200 && statusCode < 300) return 'status-success';
    if (statusCode >= 400) return 'status-error';
    return 'status-warning';
  }

  function isJSON(data) {
    // If it's already an object (parsed JSON), it's JSON
    if (typeof data === 'object' && data !== null) {
      return true;
    }
    
    // If it's a string, try to parse it
    if (typeof data === 'string') {
      try {
        JSON.parse(data);
        return true;
      } catch {
        return false;
      }
    }
    
    return false;
  }

  function handleWordWrapChange() {
    dispatch('wordWrapChange', { wordWrap });
  }

  // Helper function to safely get response body as string
  function getResponseBodyAsString(body) {
    if (body === null || body === undefined) {
      return '';
    }
    if (typeof body === 'string') {
      return body;
    }
    if (typeof body === 'object') {
      return JSON.stringify(body, null, 2);
    }
    return String(body);
  }

  async function highlightCode(codeString, language = 'json') {
    if (!codeString) return codeString;
    
    await loadHighlight();
    
    if (!hljs) return codeString;
    
    try {
      return hljs.default.highlight(codeString, { language }).value;
    } catch {
      return codeString;
    }
  }

  function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
      alert('Copied to clipboard!');
    }).catch(() => {
      alert('Failed to copy to clipboard');
    });
  }

  let lastResponseBody = '';

  // Update highlighting when response changes
  $: if (responseBodyElement && response && response.body !== undefined) {
    const currentBody = getResponseBodyAsString(response.body);
    
    if (currentBody !== lastResponseBody) {
      lastResponseBody = currentBody;
      updateHighlighting();
    }
  }

  // Handle tab switching to body tab
  $: if (activeTab === 'body' && response?.body && responseBodyElement) {

    // Use setTimeout to avoid conflicts with other reactive statements
    setTimeout(() => updateHighlighting(), 0);
  }

  async function updateHighlighting() {
    if (!responseBodyElement || !response?.body) {

      // Clear content if no response
      if (responseBodyElement) {
        responseBodyElement.textContent = '';
      }
      return;
    }
    

    
    try {
      if (responseBodyElement) {
        // Get the code element inside the pre element
        const codeElement = responseBodyElement.querySelector('code');
        
        // Get the response body as a safe string
        const bodyString = getResponseBodyAsString(response.body);
        
        // Check if content is JSON - only highlight JSON content
        if (isJSON(response.body)) {
          // Content is JSON, apply syntax highlighting
          const highlighted = await highlightCode(bodyString);
          
          if (highlighted && highlighted !== bodyString && highlighted.trim()) {
            // Highlighting worked, use it
            if (codeElement) {
              codeElement.innerHTML = highlighted;
            } else {
              responseBodyElement.innerHTML = highlighted;
            }
          } else {
            // Highlighting failed, use formatted JSON
            if (codeElement) {
              codeElement.textContent = bodyString;
            } else {
              responseBodyElement.textContent = bodyString;
            }
          }
        } else {
          // Content is NOT JSON (error messages, plain text, etc.)
          // Display as plain dark text without any highlighting
          if (codeElement) {
            codeElement.textContent = bodyString;
            // Remove any highlight.js classes and add plain text class
            codeElement.className = 'plain-text';
            codeElement.style.color = '#212529';
            codeElement.style.backgroundColor = 'transparent';
          } else {
            responseBodyElement.textContent = bodyString;
            responseBodyElement.className = 'plain-text';
            responseBodyElement.style.color = '#212529';
            responseBodyElement.style.backgroundColor = 'transparent';
          }
        }
      }
    } catch (error) {
      console.warn('Syntax highlighting failed:', error);
      // Fallback to plain text formatting
      if (responseBodyElement) {
        const codeElement = responseBodyElement.querySelector('code');
        const textContent = getResponseBodyAsString(response.body);
        
        if (codeElement) {
          codeElement.textContent = textContent;
          codeElement.className = 'plain-text';
          codeElement.style.color = '#212529';
          codeElement.style.backgroundColor = 'transparent';
        } else {
          responseBodyElement.textContent = textContent;
          responseBodyElement.className = 'plain-text';
          responseBodyElement.style.color = '#212529';
          responseBodyElement.style.backgroundColor = 'transparent';
        }
      }
    }
  }
</script>

<div class="card">
  <h2>📨 Response</h2>
  
  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
      <p>Sending request...</p>
    </div>
  {:else if response}
    <div class="response-content">
      <!-- Status -->
      <div class="response-status">
        <div class="status-info">
          <span class={`status-badge ${getStatusClass(response.statusCode)}`}>
            {response.status}
          </span>
          {#if response.error}
            <div class="error-message">
              ❌ {response.error}
            </div>
          {/if}
        </div>
      </div>

      {#if !response.error}
        <!-- Tabs Interface -->
        <div class="tabs-container">
          <div class="tabs-header">
            {#each tabs as tab}
              <button 
                type="button"
                class="tab-button"
                class:active={activeTab === tab.id}
                on:click={() => activeTab = tab.id}
              >
                {tab.icon} {tab.label}
                {#if tab.id === 'headers' && response.headers}
                  <span class="tab-count">({Object.keys(response.headers).length})</span>
                {/if}
              </button>
            {/each}
          </div>

          <div class="tab-content">
            {#if activeTab === 'body' && response.body}
              <div class="tab-panel">
                <div class="body-header">
                  <div class="body-info">
                    <span class="body-size">{new Blob([getResponseBodyAsString(response.body)]).size} bytes</span>
                    {#if isJSON(response.body)}
                      <span class="format-indicator">JSON</span>
                    {/if}
                  </div>
                  <div class="body-actions">
                    <label class="wrap-toggle">
                      <input 
                        type="checkbox" 
                        bind:checked={wordWrap}
                        on:change={handleWordWrapChange}
                      />
                      Word Wrap
                    </label>
                    <button 
                      class="btn-small" 
                      on:click={() => copyToClipboard(getResponseBodyAsString(response.body))}
                    >
                      📋 Copy
                    </button>
                  </div>
                </div>
                <div class="response-body" class:word-wrap={wordWrap}>
                  <pre bind:this={responseBodyElement}><code></code></pre>
                </div>
              </div>
            {:else if activeTab === 'body'}
              <div class="tab-panel">
                <div class="empty-state">
                  <div class="empty-icon">📄</div>
                  <p>No response body</p>
                </div>
              </div>
            {/if}

            {#if activeTab === 'headers'}
              <div class="tab-panel">
                {#if response.headers && Object.keys(response.headers).length > 0}
                  <div class="headers-grid">
                    {#each Object.entries(response.headers) as [key, value]}
                      <div class="header-item">
                        <strong>{key}:</strong> <span>{value}</span>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-state">
                    <div class="empty-icon">📋</div>
                    <p>No response headers</p>
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  {:else}
    <div class="empty-state">
      <div class="empty-icon">🚀</div>
      <p>Send a request to see the response here</p>
    </div>
  {/if}
</div>

<style>
  h2 {
    margin: 0 0 1.5rem 0;
    color: var(--text-primary, #374151);
    font-size: 1.25rem;
  }



  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 3rem;
    color: var(--text-secondary, #6b7280);
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid #f3f4f6;
    border-top: 3px solid #667eea;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 1rem;
  }

  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  .response-content {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .response-status {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }

  .status-badge {
    display: inline-block;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    font-weight: 600;
    font-size: 0.75rem;
  }

  .error-message {
    margin-top: 0.5rem;
    padding: 0.75rem;
    background: #fef2f2;
    color: #dc2626;
    border-radius: 6px;
    border: 1px solid #fecaca;
  }

  .headers-grid {
    display: grid;
    gap: 0.5rem;
    background: #f9fafb;
    padding: 1rem;
    border-radius: 6px;
    max-height: 400px;
    overflow-y: auto;
  }

  .header-item {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.75rem;
    padding: 0.5rem 0;
    border-bottom: 1px solid #e5e7eb;
    display: flex;
    justify-content: space-between;
    align-items: start;
    gap: 1rem;
  }

  .header-item:last-child {
    border-bottom: none;
  }

  .header-item strong {
    color: var(--text-primary, #374151);
    min-width: 120px;
    flex-shrink: 0;
  }

  .header-item span {
    color: var(--text-secondary, #6b7280);
    word-break: break-all;
  }

  .btn-small {
    background: var(--button-secondary, #f3f4f6);
    color: var(--text-primary, #374151);
    border: 1px solid var(--border-primary, #d1d5db);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
    transition: all 0.2s ease;
  }

  .btn-small:hover {
    background: #e5e7eb;
  }

  .format-indicator {
    background: #e0e7ff;
    color: #3730a3;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 500;
  }

  /* Tabs Styles */
  .tabs-container {
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    overflow: hidden;
  }

  .tabs-header {
    display: flex;
    background: #f9fafb;
    border-bottom: 1px solid #e5e7eb;
  }

  .tab-button {
    flex: 1;
    padding: 0.75rem;
    border: none;
    background: transparent;
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 500;
    color: #6b7280;
    transition: all 0.2s ease;
    border-bottom: 3px solid transparent;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
  }

  .tab-button:hover {
    background: var(--bg-tertiary, #f3f4f6);
    color: var(--text-primary, #374151);
  }

  .tab-button.active {
    background: var(--bg-primary, white);
    color: var(--text-accent, #667eea);
    border-bottom-color: var(--border-accent, #667eea);
  }

  .tab-count {
    background: var(--bg-tertiary, #e5e7eb);
    color: var(--text-secondary, #6b7280);
    padding: 0.125rem 0.375rem;
    border-radius: 10px;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .tab-button.active .tab-count {
    background: #e0e7ff;
    color: #667eea;
  }

  .tab-content {
    background: white;
  }

  .tab-panel {
    padding: 1rem;
  }

  .body-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    flex-wrap: wrap;
    gap: 1rem;
  }

  .body-info {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .body-size {
    background: #f3f4f6;
    color: #374151;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 500;
  }

  .body-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .wrap-toggle {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.75rem;
    color: #374151;
    cursor: pointer;
  }

  .wrap-toggle input[type="checkbox"] {
    accent-color: #667eea;
  }

  .response-body {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    overflow: auto;
    max-height: 500px;
    max-width: 100%;
  }

  .response-body pre {
    margin: 0;
    padding: 1rem;
    overflow: auto;
    white-space: pre;
    max-width: 100%;
  }

  .response-body.word-wrap pre {
    white-space: pre-wrap;
    word-wrap: break-word;
    overflow-wrap: break-word;
  }

  .response-body code {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.75rem;
    line-height: 1.5;
  }

  .response-body.word-wrap code {
    word-break: break-all;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 4rem 2rem;
    color: var(--text-secondary, #6b7280);
    text-align: center;
  }

  .empty-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
    opacity: 0.5;
  }

  /* Syntax highlighting adjustments */
  :global(.hljs) {
    background: transparent !important;
    padding: 0 !important;
  }

  /* Highlight.js token styles - light theme */
  :global(.hljs-punctuation) {
    color: #24292e !important;
    opacity: 1 !important;
  }
  
  :global(.hljs-attr) {
    color: #6f42c1 !important;
    opacity: 1 !important;
  }
  
  :global(.hljs-string) {
    color: #032f62 !important;
    opacity: 1 !important;
  }
  
  :global(.hljs-number) {
    color: #005cc5 !important;
    opacity: 1 !important;
  }
  
  :global(.hljs-literal) {
    color: #005cc5 !important;
    opacity: 1 !important;
  }
  
  :global(.hljs-keyword) {
    color: #d73a49 !important;
    opacity: 1 !important;
  }

  /* Dark theme overrides - must come after the above styles */
  :global([data-theme="dark"] .hljs-punctuation) {
    color: #d1d5db !important; /* Light gray for brackets, braces, commas */
  }
  
  :global([data-theme="dark"] .hljs-attr) {
    color: #60a5fa !important; /* Light blue for JSON keys */
  }
  
  :global([data-theme="dark"] .hljs-string) {
    color: #86efac !important; /* Light green for string values */
  }
  
  :global([data-theme="dark"] .hljs-number) {
    color: #fbbf24 !important; /* Light yellow for numbers */
  }
  
  :global([data-theme="dark"] .hljs-literal) {
    color: #a78bfa !important; /* Light purple for true/false/null */
  }
  
  :global([data-theme="dark"] .hljs-keyword) {
    color: #f87171 !important; /* Light red for keywords */
  }

  /* Enable text selection for response content */
  .response-body, 
  .response-body pre, 
  .response-body code,
  .response-body *,
  .tab-content,
  .tab-panel,
  .headers-grid,
  .headers-grid *,
  .header-item,
  .header-item *,
  .header-item strong,
  .header-item span {
    user-select: text !important;
    -webkit-user-select: text !important;
    -moz-user-select: text !important;
    cursor: text !important;
  }

  /* Specifically enable selection for highlighted code */
  :global(.hljs),
  :global(.hljs *),
  :global(.hljs-punctuation),
  :global(.hljs-attr),
  :global(.hljs-string),
  :global(.hljs-number),
  :global(.hljs-literal),
  :global(.hljs-keyword) {
    user-select: text !important;
    -webkit-user-select: text !important;
    -moz-user-select: text !important;
  }

  /* Response Headers styling */
  .headers-grid {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 0.8rem;
  }

  .header-item {
    display: flex;
    align-items: flex-start;
    padding: 0.5rem;
    background: #f8f9fa;
    border: 1px solid #e9ecef;
    border-radius: 4px;
    line-height: 1.4;
  }

  .header-item strong {
    color: #495057 !important;
    font-weight: 600;
    margin-right: 0.5rem;
    min-width: fit-content;
    flex-shrink: 0;
  }

  .header-item span {
    color: #212529 !important;
    word-break: break-all;
    flex: 1;
  }

  /* Ensure plain text responses are always readable */
  .response-body pre,
  .response-body code {
    color: #212529;
    background: #f8f9fa;
  }

  /* Override any highlight.js styles for plain text */
  .response-body code:not(.hljs) {
    color: #212529 !important;
    background: transparent !important;
  }

  /* Specific styling for plain text content (error messages, etc.) */
  .plain-text {
    color: #212529 !important;
    background: transparent !important;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace !important;
    white-space: pre-wrap !important;
    word-wrap: break-word !important;
  }

  /* Ensure plain text is not affected by any highlight.js styling */
  :global(.plain-text *) {
    color: inherit !important;
    background: transparent !important;
  }

  /* Dark theme overrides */
  :global([data-theme="dark"]) .tab-content {
    background: var(--bg-secondary);
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .tab-panel {
    background: var(--bg-secondary);
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .tab-button {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .tab-button.active {
    background: var(--bg-secondary);
    color: var(--text-accent);
    border-bottom-color: var(--border-accent);
  }

  :global([data-theme="dark"]) .tab-count {
    background: var(--bg-accent);
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .tab-button.active .tab-count {
    background: var(--border-accent);
    color: white;
  }

  :global([data-theme="dark"]) .body-header,
  :global([data-theme="dark"]) .headers-header {
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .response-body {
    background: var(--bg-primary);
    color: var(--text-primary);
    border-color: var(--border-primary);
  }

  :global([data-theme="dark"]) .response-body pre {
    background: var(--bg-primary);
    color: #e5e7eb !important;
  }

  :global([data-theme="dark"]) .response-body code {
    background: var(--bg-primary);
    color: #e5e7eb !important;
  }

  :global([data-theme="dark"]) .response-body.word-wrap pre {
    background: var(--bg-primary);
    color: #e5e7eb !important;
  }

  :global([data-theme="dark"]) .response-body.word-wrap code {
    background: var(--bg-primary);
    color: #e5e7eb !important;
  }

  /* Ensure syntax highlighting doesn't override our dark theme colors */
  :global([data-theme="dark"]) .response-body pre *,
  :global([data-theme="dark"]) .response-body code * {
    color: #e5e7eb !important;
  }







  :global([data-theme="dark"]) .body-info,
  :global([data-theme="dark"]) .body-actions {
    color: var(--text-secondary);
  }

  :global([data-theme="dark"]) .body-size,
  :global([data-theme="dark"]) .format-indicator {
    background: var(--bg-accent);
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .wrap-toggle {
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .btn-small {
    background: var(--button-secondary);
    color: var(--text-primary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .btn-small:hover {
    background: var(--button-secondary-hover);
    border-color: var(--border-accent);
  }

  :global([data-theme="dark"]) .status-info {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .loading-spinner {
    border-color: var(--border-secondary);
    border-top-color: var(--text-accent);
  }

  /* Dark theme overrides for plain text responses */
  :global([data-theme="dark"]) .plain-text {
    color: #e5e7eb !important;
    background: transparent !important;
  }

  /* Override hardcoded light theme colors for plain text in dark mode */
  :global([data-theme="dark"]) .response-body pre:not(.hljs),
  :global([data-theme="dark"]) .response-body code:not(.hljs) {
    color: #e5e7eb !important;
    background: var(--bg-primary) !important;
  }
</style>