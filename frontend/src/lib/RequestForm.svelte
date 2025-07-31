<script>
  import { createEventDispatcher, onMount } from 'svelte';
  
  const dispatch = createEventDispatcher();
  export let loading = false;
  export let selectedRequest = null;
  export let canSend = false;
  export let variables = []; // Add variables prop

  let url = '';
  let method = 'GET';
  let headers = '{\n  "Content-Type": "application/json"\n}';
  let body = '';
  let headersValid = true;
  let activeTab = 'headers';
  let params = [{ key: '', value: '', enabled: true }];
  let saveStatus = ''; // Track save status for user feedback


  const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];
  const tabs = [
    { id: 'headers', label: 'Headers', icon: '📋' },
    { id: 'params', label: 'Params', icon: '🔍' }
  ];

  function validateHeaders() {
    try {
      JSON.parse(headers);
      headersValid = true;
    } catch {
      headersValid = false;
    }
  }

  // Process template variables in a string (frontend version)
  function processTemplate(input, vars) {
    if (!input || !vars) return input;
    
    let result = input;
    vars.forEach(variable => {
      if (variable.key && variable.value !== undefined) {
        const regex = new RegExp(`{{\\s*${variable.key}\\s*}}`, 'g');
        result = result.replace(regex, variable.value);
      }
    });
    return result;
  }

  // Check if URL is valid or contains template variables
  function isValidUrlOrTemplate(urlString) {
    if (!urlString || !urlString.trim()) return false;
    
    // If it contains template variables, consider it valid for now
    if (urlString.includes('{{') && urlString.includes('}}')) {
      return true;
    }
    
    // Otherwise, check if it's a valid URL
    try {
      new URL(urlString);
      return true;
    } catch {
      return false;
    }
  }

  function addParam() {
    params = [...params, { key: '', value: '', enabled: true }];
    updatePreview();
  }

  function removeParam(index) {
    params = params.filter((_, i) => i !== index);
    updatePreview();
  }

  function buildUrlWithParams() {
    if (!url || !url.trim()) {
      return '';
    }
    
    // First, process template variables in the URL
    const processedUrl = processTemplate(url.trim(), variables);
    
    try {
      // Try to create a URL object with the processed URL
      const urlObj = new URL(processedUrl);
      
      // Clear existing search params
      urlObj.search = '';
      
      // Add our params (also process templates in param values)
      const enabledParams = params.filter(param => param.enabled && param.key && param.key.trim());
      
      enabledParams.forEach(param => {
        const key = processTemplate(param.key.trim(), variables);
        const value = processTemplate(param.value ? param.value.trim() : '', variables);
        urlObj.searchParams.append(key, value);
      });
      
      return urlObj.toString();
    } catch (error) {
      // If URL is invalid after processing, try to build it manually for templates
      if (url.includes('{{') && url.includes('}}')) {
        // For template URLs, append params manually
        let result = processedUrl;
        const enabledParams = params.filter(param => param.enabled && param.key && param.key.trim());
        
        if (enabledParams.length > 0) {
          const queryParams = enabledParams.map(param => {
            const key = processTemplate(param.key.trim(), variables);
            const value = processTemplate(param.value ? param.value.trim() : '', variables);
            return `${encodeURIComponent(key)}=${encodeURIComponent(value)}`;
          }).join('&');
          
          result += (result.includes('?') ? '&' : '?') + queryParams;
        }
        
        return result;
      }
      
      // If it's not a template and still invalid, return as-is
      console.warn('Invalid URL:', processedUrl, error);
      return processedUrl;
    }
  }

  let previewUrl = '';

  function updatePreview() {
    previewUrl = buildUrlWithParams();
  }

  // Initialize preview on component mount and when variables/URL change
  $: if (url || variables) updatePreview();

  // Also update when params change
  $: if (params) updatePreview();

  // Debounced save when form changes
  let urlSaveTimeout;
  let isLoadingRequest = false; // Flag to prevent save during loading
  
  // Track all form state for change detection
  let lastSavedState = {
    url: '',
    method: 'GET',
    headers: '',
    body: '',
    params: []
  };
  
  // Check if any form data has changed
  function hasFormChanged() {
    if (!selectedRequest) return false;
    
    const currentParams = params.filter(p => p.key && p.key.trim());
    const lastParams = lastSavedState.params || [];
    
    return (
      url !== lastSavedState.url ||
      method !== lastSavedState.method ||
      headers !== lastSavedState.headers ||
      body !== lastSavedState.body ||
      JSON.stringify(currentParams) !== JSON.stringify(lastParams)
    );
  }

  $: {
    console.log('🔍 Reactive statement triggered:');
    console.log('  - selectedRequest:', selectedRequest?.name || 'none');
    console.log('  - url:', url);
    console.log('  - method:', method);
    console.log('  - isLoadingRequest:', isLoadingRequest);
    console.log('  - hasFormChanged:', hasFormChanged());
    
    if (selectedRequest && url && url.trim() && hasFormChanged() && !isLoadingRequest) {
      clearTimeout(urlSaveTimeout);
      saveStatus = 'pending';
      
      console.log('🔄 Form changed, will auto-save in 2 seconds');
      
      urlSaveTimeout = setTimeout(() => {
        console.log('⏰ Auto-save timeout triggered');
        saveCurrentRequest();
      }, 2000); // Longer delay for auto-save, manual save button is primary
    } else {
      console.log('❌ Auto-save skipped - reason:');
      if (!selectedRequest) console.log('  - No selected request');
      if (!url || !url.trim()) console.log('  - URL is empty');
      if (!hasFormChanged()) console.log('  - No changes detected');
      if (isLoadingRequest) console.log('  - Currently loading request');
    }
  }

  // Save immediately when URL field loses focus
  function handleUrlBlur() {
    if (selectedRequest && url && url.trim() && hasFormChanged()) {
      console.log('👆 URL field lost focus, saving immediately:', url);
      clearTimeout(urlSaveTimeout);
      saveCurrentRequest();
    }
  }

  function handleSubmit() {
    if (!canSend) {
      alert('Please select a request from the collection first');
      return;
    }

    if (!url.trim()) {
      alert('URL is required');
      return;
    }

    // Validate URL or template
    if (!isValidUrlOrTemplate(url.trim())) {
      alert('Please enter a valid URL');
      return;
    }

    if (!headersValid) {
      alert('Headers must be valid JSON');
      return;
    }

    let parsedHeaders = {};
    try {
      parsedHeaders = JSON.parse(headers);
    } catch {
      parsedHeaders = {};
    }

    // For API requests, we DO want to use the processed URL with variables substituted
    const processedUrl = buildUrlWithParams();
    const requestData = {
      url: processedUrl || url.trim(),
      method,
      headers: parsedHeaders,
      body: body.trim(),
      params: params.filter(p => p.key && p.key.trim())
    };
    
    console.log('📤 Sending request:', method, requestData.url);
    
    // Save the request when sending (this will save the RAW URL)
    saveCurrentRequest();
    
    dispatch('submit', requestData);
  }

  // Manual save function
  function handleManualSave() {
    if (!selectedRequest || !url.trim()) {
      console.log('❌ Manual save aborted - missing selectedRequest or URL');
      return;
    }
    console.log('👆 Manual save button clicked');
    saveCurrentRequest();
  }

  // Save current request data
  function saveCurrentRequest() {
    console.log('💾 saveCurrentRequest called');
    console.log('🔍 selectedRequest:', selectedRequest?.name || 'none');
    console.log('🔍 url:', url);
    
    if (!selectedRequest || !url.trim()) {
      console.log('❌ Save aborted - missing selectedRequest or URL');
      return;
    }
    
    let parsedHeaders = {};
    try {
      parsedHeaders = JSON.parse(headers);
    } catch {
      parsedHeaders = {};
    }

    // Clear any pending URL save timeout since we're saving now
    clearTimeout(urlSaveTimeout);
    saveStatus = 'saving';

    // Save the RAW URL that user typed, not the processed template URL
    // Template processing should only happen during API requests, not when saving
    const requestData = {
      url: url.trim(), // Save raw URL with template variables intact
      method,
      headers: parsedHeaders,
      body: body.trim(),
      params: params.filter(p => p.key && p.key.trim())
    };

    console.log('💾 Saving request changes...');
    console.log('🔍 Current URL in form:', url);
    console.log('🔍 Method:', method);
    console.log('🔍 Final URL to save (RAW):', requestData.url);
    console.log('🔍 Selected request ID:', selectedRequest.id);
    console.log('📤 Dispatching save event with data:', requestData);
    
    // Update our saved state tracking
    lastSavedState = {
      url: url,
      method: method,
      headers: headers,
      body: body,
      params: [...params.filter(p => p.key && p.key.trim())]
    };
    
    dispatch('save', requestData);

    // Clear save status after a short delay
    setTimeout(() => {
      saveStatus = 'saved';
      setTimeout(() => {
        saveStatus = '';
      }, 1500);
    }, 200);
  }

  function addCommonHeader(header) {
    try {
      const currentHeaders = JSON.parse(headers);
      let newHeader = {};
      
      switch(header) {
        case 'auth':
          newHeader = { "Authorization": "Bearer YOUR_TOKEN_HERE" };
          break;
        case 'json':
          newHeader = { "Content-Type": "application/json" };
          break;
        case 'form':
          newHeader = { "Content-Type": "application/x-www-form-urlencoded" };
          break;
      }
      
      const merged = { ...currentHeaders, ...newHeader };
      headers = JSON.stringify(merged, null, 2);
      validateHeaders();
    } catch {
      // If headers aren't valid JSON, replace them
      headers = JSON.stringify(newHeader, null, 2);
      validateHeaders();
    }
  }

  $: validateHeaders();



  function loadRequestData(event) {
    const data = event.detail;
    
    console.log('📥 Loading request data:', data);
    
    // Set loading flag to prevent auto-save during load
    isLoadingRequest = true;
    
    // Clear any pending saves and status
    clearTimeout(urlSaveTimeout);
    saveStatus = '';
    
    // Populate form fields
    const newUrl = data.url || '';
    console.log('📝 Setting URL from:', url, 'to:', newUrl);
    
    url = newUrl;
    method = data.method || 'GET';
    body = data.body || '';
    
    // Handle headers
    if (data.headers && Object.keys(data.headers).length > 0) {
      headers = JSON.stringify(data.headers, null, 2);
    } else {
      headers = '{\n  "Content-Type": "application/json"\n}';
    }
    
    // Handle params
    if (data.params && data.params.length > 0) {
      params = data.params.map(p => ({
        key: p.key || '',
        value: p.value || '',
        enabled: p.enabled !== false
      }));
    } else {
      params = [{ key: '', value: '', enabled: true }];
    }
    
    // Update preview and validate
    updatePreview();
    validateHeaders();
    
    // Initialize saved state tracking
    lastSavedState = {
      url: newUrl,
      method: method,
      headers: headers,
      body: body,
      params: [...params.filter(p => p.key && p.key.trim())]
    };
    
    // Clear loading flag after a short delay to allow reactive statements to settle
    setTimeout(() => {
      isLoadingRequest = false;
      console.log('✅ Request data loaded, auto-save re-enabled');
    }, 100);
    
    console.log('📥 Loaded request data complete');
  }

  onMount(() => {
    // Listen for loadRequest events from the parent
    window.addEventListener('loadRequest', loadRequestData);
    
    return () => {
      window.removeEventListener('loadRequest', loadRequestData);
      // Clean up timeout to prevent memory leaks
      clearTimeout(urlSaveTimeout);
    };
  });

  // Export function so parent can call it when switching requests
  export { saveCurrentRequest };
</script>

<div class="card">
  <h2>🔄 HTTP Request</h2>
  
  <form on:submit|preventDefault={handleSubmit}>
    <div class="form-group">
      <div class="url-label-container">
        <label for="url">URL *</label>
        {#if saveStatus === 'pending'}
          <span class="save-status pending">✏️ Changes pending...</span>
        {:else if saveStatus === 'saving'}
          <span class="save-status saving">💾 Saving...</span>
        {:else if saveStatus === 'saved'}
          <span class="save-status saved">✅ Saved</span>
        {/if}
      </div>
      <input 
        id="url"
        type="text"
        bind:value={url} 
        on:blur={handleUrlBlur}
        placeholder="https://api.example.com/endpoint or {'{'}{'{'}}host{'}'}{'}'}}/api/endpoint"
        class="input"
        required
      />
    </div>

    <div class="form-group">
      <label for="method">Method</label>
      <select id="method" bind:value={method} class="select">
        {#each methods as methodOption}
          <option value={methodOption}>{methodOption}</option>
        {/each}
      </select>
    </div>

    <!-- Preview URL -->
    <div class="form-group">
      <div class="url-preview-top">
        <span>Preview URL:</span>
        {#if previewUrl !== url}
          <div class="preview-info">
            <small class="template-info">✨ Variables substituted</small>
          </div>
        {/if}
        <div class="preview-text-top">{previewUrl}</div>
      </div>
    </div>

    <!-- Tabs Interface -->
    <div class="form-group">
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
            </button>
          {/each}
        </div>

        <div class="tab-content">
          {#if activeTab === 'headers'}
            <div class="tab-panel">
              <div class="headers-label">
                <label for="headers">Headers (JSON)</label>
                <div class="header-shortcuts">
                  <button type="button" class="btn-small" on:click={() => addCommonHeader('json')}>+ JSON</button>
                  <button type="button" class="btn-small" on:click={() => addCommonHeader('form')}>+ Form</button>
                  <button type="button" class="btn-small" on:click={() => addCommonHeader('auth')}>+ Auth</button>
                </div>
              </div>
              <textarea 
                id="headers"
                bind:value={headers} 
                placeholder="Headers in JSON format"
                class="textarea"
                class:invalid={!headersValid}
                rows="6"
              ></textarea>
              {#if !headersValid}
                <div class="error-message">Invalid JSON format</div>
              {/if}
            </div>
          {/if}

          {#if activeTab === 'params'}
            <div class="tab-panel">
              <div class="params-header">
                <span>URL Query Parameters</span>
                <button type="button" class="btn-small" on:click={addParam}>+ Add Param</button>
              </div>
              
              <div class="params-list">
                {#each params as param, index}
                  <div class="param-row">
                    <input 
                      type="checkbox" 
                      bind:checked={param.enabled}
                      class="param-checkbox"
                      on:change={updatePreview}
                    />
                    <input 
                      type="text" 
                      bind:value={param.key}
                      placeholder="Key"
                      class="input param-input"
                      on:input={updatePreview}
                    />
                    <input 
                      type="text" 
                      bind:value={param.value}
                      placeholder="Value"
                      class="input param-input"
                      on:input={updatePreview}
                    />
                    <button 
                      type="button" 
                      class="btn-remove"
                      on:click={() => removeParam(index)}
                      disabled={params.length === 1}
                    >
                      ❌
                    </button>
                  </div>
                {/each}
              </div>


            </div>
          {/if}
        </div>
      </div>
    </div>

    {#if method !== 'GET' && method !== 'HEAD'}
      <div class="form-group">
        <label for="body">Request Body</label>
        <textarea 
          id="body"
          bind:value={body} 
          placeholder={method === 'POST' ? '{"key": "value"}' : ''}
          class="textarea"
          rows="6"
        ></textarea>
      </div>
    {/if}

    <div class="button-row">
      <button 
        type="button" 
        class="btn btn-secondary" 
        disabled={!canSend || !hasFormChanged()}
        on:click={handleManualSave}
        title="Save changes to this request"
      >
        {#if saveStatus === 'saving'}
          💾 Saving...
        {:else if saveStatus === 'saved'}
          ✅ Saved
        {:else if hasFormChanged()}
          💾 Save Changes
        {:else}
          ✅ No Changes
        {/if}
      </button>

      <button type="submit" class="btn btn-primary" disabled={loading || !headersValid || !canSend}>
        {#if loading}
          🔄 Sending...
        {:else if !canSend}
          📤 Select a Request First
        {:else}
          📤 Send Request
        {/if}
      </button>
    </div>
    
    {#if !canSend && !loading}
      <p class="send-hint">Select a request from the collection to enable sending</p>
    {/if}
    
    {#if hasFormChanged() && canSend}
      <p class="changes-hint">💡 You have unsaved changes. Click "Save Changes" or they will auto-save in 2 seconds.</p>
    {/if}
  </form>
</div>

<style>
  h2 {
    margin: 0 0 1.5rem 0;
    color: #374151;
    font-size: 1.5rem;
  }

  .headers-label {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .header-shortcuts {
    display: flex;
    gap: 0.5rem;
  }

  .btn-small {
    background: #f3f4f6;
    color: #374151;
    border: 1px solid #d1d5db;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.875rem;
    transition: all 0.2s ease;
  }

  .btn-small:hover {
    background: #e5e7eb;
  }

  .button-row {
    display: flex;
    gap: 1rem;
    align-items: stretch;
  }

  .btn-primary {
    flex: 2;
    font-size: 1.1rem;
    padding: 1rem;
  }

  .btn-secondary {
    flex: 1;
    background: #f3f4f6;
    color: #374151;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    padding: 1rem;
    font-size: 0.9rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 48px;
  }

  .btn-secondary:hover:not(:disabled) {
    background: #e5e7eb;
    border-color: #9ca3af;
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  }

  .btn-secondary:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none;
    box-shadow: none;
  }

  .invalid {
    border-color: #ef4444 !important;
    background-color: #fef2f2;
  }

  .error-message {
    color: #ef4444;
    font-size: 0.875rem;
    margin-top: 0.25rem;
  }

  textarea {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.9rem;
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
    padding: 1rem;
    border: none;
    background: transparent;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: 500;
    color: #6b7280;
    transition: all 0.2s ease;
    border-bottom: 3px solid transparent;
  }

  .tab-button:hover {
    background: #f3f4f6;
    color: #374151;
  }

  .tab-button.active {
    background: white;
    color: #667eea;
    border-bottom-color: #667eea;
  }

  .tab-content {
    background: white;
  }

  .tab-panel {
    padding: 1.5rem;
  }

  /* Params Styles */
  .params-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .params-header span {
    font-weight: 500;
    color: #374151;
    margin: 0;
  }

  .params-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .param-row {
    display: grid;
    grid-template-columns: auto 1fr 1fr auto;
    gap: 0.75rem;
    align-items: center;
    padding: 0.75rem;
    background: #f9fafb;
    border-radius: 6px;
    border: 1px solid #e5e7eb;
  }

  .param-checkbox {
    width: 18px;
    height: 18px;
    accent-color: #667eea;
  }

  .param-input {
    margin: 0;
    font-size: 0.9rem;
  }

  .btn-remove {
    background: #fef2f2;
    color: #dc2626;
    border: 1px solid #fecaca;
    padding: 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.875rem;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
  }

  .btn-remove:hover:not(:disabled) {
    background: #fee2e2;
  }

  .btn-remove:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .url-preview-top {
    padding: 1rem;
    background: #f0f9ff;
    border: 1px solid #bae6fd;
    border-radius: 6px;
  }

  .url-preview-top span {
    display: block;
    font-weight: 500;
    color: #0369a1;
    margin-bottom: 0.5rem;
    font-size: 0.875rem;
  }

  .preview-info {
    margin-bottom: 0.5rem;
  }

  .template-info {
    color: #059669;
    font-size: 0.75rem;
    font-weight: 500;
    background: #d1fae5;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    border: 1px solid #a7f3d0;
  }

  .preview-text-top {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.875rem;
    color: #0c4a6e;
    background: white;
    padding: 0.75rem;
    border-radius: 4px;
    border: 1px solid #bae6fd;
    word-break: break-all;
    max-height: 100px;
    overflow-y: auto;
  }

  .send-hint {
    margin-top: 0.5rem;
    font-size: 0.875rem;
    color: #6b7280;
    text-align: center;
    font-style: italic;
  }

  .changes-hint {
    margin-top: 0.5rem;
    font-size: 0.875rem;
    color: #d97706;
    text-align: center;
    background: #fef3c7;
    padding: 0.75rem;
    border-radius: 6px;
    border: 1px solid #f59e0b;
  }

  /* URL Save Status Styles */
  .url-label-container {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .save-status {
    font-size: 0.75rem;
    font-weight: 500;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    transition: all 0.2s ease;
  }

  .save-status.pending {
    background: #fef3c7;
    color: #d97706;
    border: 1px solid #f59e0b;
  }

  .save-status.saving {
    background: #dbeafe;
    color: #2563eb;
    border: 1px solid #3b82f6;
    animation: pulse 1.5s ease-in-out infinite;
  }

  .save-status.saved {
    background: #dcfce7;
    color: #16a34a;
    border: 1px solid #22c55e;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.6; }
  }
</style>