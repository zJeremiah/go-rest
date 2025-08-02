<script>
  import { createEventDispatcher, onMount } from 'svelte';
  
  const dispatch = createEventDispatcher();
  export let loading = false;
  export let selectedRequest = null;
  export let canSend = false;
  export let variables = []; // Add variables prop
  export let groups = []; // Groups for selection

  let url = '';
  let method = 'GET';
  let headers = [{ key: 'Content-Type', value: 'application/json', enabled: true }];
  let body = '';
  let activeTab = 'headers';
  let params = [{ key: '', value: '', enabled: true }];
  let saveStatus = ''; // Track save status for user feedback


  const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];
  const tabs = [
    { id: 'headers', label: 'Headers', icon: '📋' },
    { id: 'params', label: 'Params', icon: '🔍' }
  ];

  function addHeader() {
    headers = [...headers, { key: '', value: '', enabled: true }];
  }

  function removeHeader(index) {
    headers = headers.filter((_, i) => i !== index);
  }

  // Build headers with variable substitution for preview
  function buildHeadersPreview() {
    const processedHeaders = {};
    headers.filter(h => h.enabled && h.key && h.key.trim()).forEach(header => {
      const processedKey = processTemplate(header.key.trim(), variables);
      const processedValue = processTemplate(header.value || '', variables);
      processedHeaders[processedKey] = processedValue;
    });
    return processedHeaders;
  }

  // Check if any headers contain variables
  function hasHeaderVariables() {
    return headers.some(h => 
      h.enabled && h.key && 
      ((h.key.includes('{{') && h.key.includes('}}')) || 
       (h.value && h.value.includes('{{') && h.value.includes('}}')))
    );
  }

  let previewHeaders = {};
  
  function updateHeadersPreview() {
    previewHeaders = buildHeadersPreview();
  }

  // Update headers preview when headers or variables change
  $: if (headers || variables) updateHeadersPreview();

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
    
    const currentHeaders = headers.filter(h => h.key && h.key.trim());
    const lastHeaders = lastSavedState.headers || [];
    
    return (
      url !== lastSavedState.url ||
      method !== lastSavedState.method ||
      JSON.stringify(currentHeaders) !== JSON.stringify(lastHeaders) ||
      body !== lastSavedState.body ||
      JSON.stringify(currentParams) !== JSON.stringify(lastParams)
    );
  }

  // Auto-save when any form field changes (explicitly depend on all form fields)
  $: if (selectedRequest && url && url.trim() && !isLoadingRequest && (url || method || headers || body || params)) {
    // Check for changes in url, method, headers, body, params
    const formChanged = hasFormChanged();
    if (formChanged) {
      clearTimeout(urlSaveTimeout);
      saveStatus = 'pending';
      
      urlSaveTimeout = setTimeout(() => {
        saveCurrentRequest();
      }, 2000); // Longer delay for auto-save, manual save button is primary
    }
  }

  // Save immediately when URL field loses focus
  function handleUrlBlur() {
    if (selectedRequest && url && url.trim() && hasFormChanged()) {
      clearTimeout(urlSaveTimeout);
      saveCurrentRequest();
    }
  }

  // Save immediately when header/param fields lose focus
  function handleFieldBlur() {
    if (selectedRequest && url && url.trim() && hasFormChanged()) {
      clearTimeout(urlSaveTimeout);
      saveCurrentRequest();
    }
  }

  // Handle group change
  function handleGroupChange() {
    if (selectedRequest) {
      clearTimeout(urlSaveTimeout);
      // Force save immediately for group changes
      saveCurrentRequest();
      // Also trigger a manual save to be extra sure
      setTimeout(() => {
        saveCurrentRequest();
      }, 100);
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

    // Convert headers array to object
    const parsedHeaders = {};
    headers.filter(h => h.enabled && h.key && h.key.trim()).forEach(header => {
      parsedHeaders[header.key.trim()] = header.value || '';
    });

    // For API requests, we DO want to use the processed URL with variables substituted
    const processedUrl = buildUrlWithParams();
    const requestData = {
      url: processedUrl || url.trim(),
      method,
      headers: parsedHeaders,
      body: body.trim(),
      params: params.filter(p => p.key && p.key.trim())
    };
    

    
    // Save the request when sending (this will save the RAW URL)
    saveCurrentRequest();
    
    dispatch('submit', requestData);
  }

  // Manual save function
  function handleManualSave() {
    if (!selectedRequest || !url.trim()) {

      return;
    }

    saveCurrentRequest();
  }

  // Save current request data
  function saveCurrentRequest() {

    
    if (!selectedRequest || !url.trim()) {

      return;
    }
    
    // Convert headers array to object for saving
    const parsedHeaders = {};
    headers.filter(h => h.enabled && h.key && h.key.trim()).forEach(header => {
      parsedHeaders[header.key.trim()] = header.value || '';
    });

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
      params: params.filter(p => p.key && p.key.trim()),
      group: selectedRequest?.group || 'default'
    };


    
    // Update our saved state tracking
    lastSavedState = {
      url: url,
      method: method,
      headers: [...headers.filter(h => h.key && h.key.trim())],
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

  function addCommonHeader(headerType) {
    let newHeader = { key: '', value: '', enabled: true };
    
    switch(headerType) {
      case 'auth':
        newHeader = { key: 'Authorization', value: 'Bearer YOUR_TOKEN_HERE', enabled: true };
        break;
      case 'json':
        newHeader = { key: 'Content-Type', value: 'application/json', enabled: true };
        break;
      case 'form':
        newHeader = { key: 'Content-Type', value: 'application/x-www-form-urlencoded', enabled: true };
        break;
    }
    
    headers = [...headers, newHeader];
  }



  function loadRequestData(event) {
    const data = event.detail;
    

    
    // Set loading flag to prevent auto-save during load
    isLoadingRequest = true;
    
    // Clear any pending saves and status
    clearTimeout(urlSaveTimeout);
    saveStatus = '';
    
    // Populate form fields
    const newUrl = data.url || '';

    
    url = newUrl;
    method = data.method || 'GET';
    body = data.body || '';
    
    // Handle headers - convert from object to array format
    if (data.headers && Object.keys(data.headers).length > 0) {
      headers = Object.entries(data.headers).map(([key, value]) => ({
        key: key,
        value: value,
        enabled: true
      }));
    } else {
      headers = [{ key: 'Content-Type', value: 'application/json', enabled: true }];
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
    
    // Update preview
    updatePreview();
    
    // Initialize saved state tracking
    lastSavedState = {
      url: newUrl,
      method: method,
      headers: [...headers.filter(h => h.key && h.key.trim())],
      body: body,
      params: [...params.filter(p => p.key && p.key.trim())]
    };
    
    // Clear loading flag after a short delay to allow reactive statements to settle
    setTimeout(() => {
      isLoadingRequest = false;
  
    }, 100);
    

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
      <select id="method" bind:value={method} on:blur={handleFieldBlur} class="select">
        {#each methods as methodOption}
          <option value={methodOption}>{methodOption}</option>
        {/each}
      </select>
    </div>

    {#if selectedRequest}
      <div class="form-group">
        <label for="group">Group</label>
        <select id="group" bind:value={selectedRequest.group} on:change={handleGroupChange} class="select">
          {#each groups as group}
            <option value={group.name}>{group.name}</option>
          {/each}
        </select>
      </div>
    {/if}

    <!-- Send Request Buttons - Moved to top for better UX -->
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

      <button type="submit" class="btn btn-primary" disabled={loading || !canSend}>
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
              <div class="headers-header">
                <span>HTTP Headers</span>
                <div class="header-shortcuts">
                  <button type="button" class="btn-small" on:click={() => addCommonHeader('json')}>+ JSON</button>
                  <button type="button" class="btn-small" on:click={() => addCommonHeader('form')}>+ Form</button>
                  <button type="button" class="btn-small" on:click={() => addCommonHeader('auth')}>+ Auth</button>
                  <button type="button" class="btn-small" on:click={addHeader}>+ Add Header</button>
                </div>
              </div>
              
              <div class="headers-list">
                {#each headers as header, index}
                  <div class="header-row">
                    <input 
                      type="checkbox" 
                      bind:checked={header.enabled}
                      class="header-checkbox"
                    />
                    <input 
                      type="text" 
                      bind:value={header.key}
                      on:blur={handleFieldBlur}
                      placeholder="Header Name"
                      class="input header-input"
                    />
                    <input 
                      type="text" 
                      bind:value={header.value}
                      on:blur={handleFieldBlur}
                      placeholder="Header Value"
                      class="input header-input"
                    />
                    <button 
                      type="button" 
                      class="btn-remove"
                      on:click={() => removeHeader(index)}
                      disabled={headers.length === 1}
                    >
                      ❌
                    </button>
                  </div>
                {/each}
              </div>

              <!-- Headers Preview -->
              {#if Object.keys(previewHeaders).length > 0}
                <div class="headers-preview">
                  <div class="preview-header">
                    <span>Preview Headers:</span>
                    {#if hasHeaderVariables()}
                      <div class="preview-info">
                        <small class="template-info">✨ Variables substituted</small>
                      </div>
                    {/if}
                  </div>
                  <div class="preview-content">
                    {#each Object.entries(previewHeaders) as [key, value]}
                      <div class="preview-header-row">
                        <span class="preview-key">{key}:</span>
                        <span class="preview-value">{value}</span>
                      </div>
                    {/each}
                  </div>
                </div>
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
                      on:blur={handleFieldBlur}
                      placeholder="Key"
                      class="input param-input"
                      on:input={updatePreview}
                    />
                    <input 
                      type="text" 
                      bind:value={param.value}
                      on:blur={handleFieldBlur}
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
          on:blur={handleFieldBlur}
          placeholder={method === 'POST' ? '{"key": "value"}' : ''}
          class="textarea"
          rows="6"
        ></textarea>
      </div>
    {/if}

  </form>
</div>

<style>
  h2 {
    margin: 0 0 1.5rem 0;
    color: var(--text-primary, #374151);
    font-size: 1.25rem;
  }



  .header-shortcuts {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .btn-small {
    background: #f3f4f6;
    color: #374151;
    border: 1px solid #d1d5db;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
    transition: all 0.2s ease;
  }

  .btn-small:hover {
    background: #e5e7eb;
  }

  .button-row {
    display: flex;
    gap: 1rem;
    align-items: stretch;
    margin-bottom: 1.5rem;
  }

  .btn-primary {
    flex: 2;
    font-size: 0.75rem;
    padding: 0.75rem 1rem;
    border: none;
    background: var(--button-primary, #3b82f6);
    color: white;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 500;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--button-primary-hover, #2563eb);
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(59, 130, 246, 0.2);
  }

  .btn-primary:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none;
    box-shadow: none;
  }

  .btn-secondary {
    flex: 1;
    background: #f3f4f6;
    color: #374151;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    padding: 0.75rem 1rem;
    font-size: 0.8rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: auto;
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



  textarea {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.8rem;
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
    font-size: 0.8rem;
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
    max-height: 400px;
    overflow-y: auto;
  }

  .tab-panel {
    padding: 1.5rem;
  }

  /* Headers Styles */
  .headers-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .headers-header span {
    font-weight: 500;
    color: #374151;
    margin: 0;
  }

  .headers-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    max-height: 300px;
    overflow-y: auto;
  }

  .header-row {
    display: grid;
    grid-template-columns: auto 1fr 1fr auto;
    gap: 0.75rem;
    align-items: center;
    padding: 0.75rem;
    background: #f9fafb;
    border-radius: 6px;
    border: 1px solid #e5e7eb;
  }

  .header-checkbox {
    width: 18px;
    height: 18px;
    accent-color: #667eea;
  }

  .header-input {
    margin: 0;
    font-size: 0.8rem;
  }

  /* Headers Preview Styles */
  .headers-preview {
    margin-top: 1rem;
    padding: 1rem;
    background: var(--preview-bg, #f0f9ff);
    border: 1px solid var(--preview-border, #bae6fd);
    border-radius: 6px;
  }

  .preview-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.75rem;
  }

  .preview-header span {
    font-weight: 500;
    color: var(--preview-text, #0369a1);
    font-size: 0.75rem;
  }

  .preview-info {
    display: flex;
    align-items: center;
  }

  .preview-content {
    background: var(--bg-primary, white);
    border: 1px solid var(--preview-border, #bae6fd);
    border-radius: 4px;
    padding: 0.75rem;
    max-height: 200px;
    overflow-y: auto;
  }

  .preview-header-row {
    display: flex;
    margin-bottom: 0.5rem;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.75rem;
  }

  .preview-header-row:last-child {
    margin-bottom: 0;
  }

  .preview-key {
    color: #1e40af;
    font-weight: 600;
    min-width: 140px;
    margin-right: 0.5rem;
  }

  .preview-value {
    color: #059669;
    word-break: break-all;
    flex: 1;
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
    max-height: 300px;
    overflow-y: auto;
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
    font-size: 0.8rem;
  }

  .btn-remove {
    background: #fef2f2;
    color: #dc2626;
    border: 1px solid #fecaca;
    padding: 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
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
    background: var(--preview-bg, #f0f9ff);
    border: 1px solid var(--preview-border, #bae6fd);
    border-radius: 6px;
  }

  .url-preview-top span {
    display: block;
    font-weight: 500;
    color: var(--preview-text, #0369a1);
    margin-bottom: 0.5rem;
    font-size: 0.75rem;
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
    font-size: 0.75rem;
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
    font-size: 0.75rem;
    color: #6b7280;
    text-align: center;
    font-style: italic;
  }

  .changes-hint {
    margin-top: 0.5rem;
    font-size: 0.75rem;
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

  /* Custom scrollbar styling */
  .tab-content::-webkit-scrollbar,
  .params-list::-webkit-scrollbar,
  .headers-list::-webkit-scrollbar,
  .preview-content::-webkit-scrollbar {
    width: 8px;
  }

  .tab-content::-webkit-scrollbar-track,
  .params-list::-webkit-scrollbar-track,
  .headers-list::-webkit-scrollbar-track,
  .preview-content::-webkit-scrollbar-track {
    background: #f1f5f9;
    border-radius: 4px;
  }

  .tab-content::-webkit-scrollbar-thumb,
  .params-list::-webkit-scrollbar-thumb,
  .headers-list::-webkit-scrollbar-thumb,
  .preview-content::-webkit-scrollbar-thumb {
    background: #cbd5e1;
    border-radius: 4px;
  }

  .tab-content::-webkit-scrollbar-thumb:hover,
  .params-list::-webkit-scrollbar-thumb:hover,
  .headers-list::-webkit-scrollbar-thumb:hover,
  .preview-content::-webkit-scrollbar-thumb:hover {
    background: #94a3b8;
  }

  /* Enable text selection for URL preview and preview headers */
  .preview-text-top,
  .url-preview-top,
  .url-preview-top *,
  .preview-content,
  .preview-header-row,
  .preview-header-row *,
  .preview-key,
  .preview-value {
    user-select: text !important;
    -webkit-user-select: text !important;
    -moz-user-select: text !important;
    cursor: text !important;
  }
</style>