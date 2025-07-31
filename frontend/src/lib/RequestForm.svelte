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

  // Save when URL changes (important changes should be saved)
  $: if (selectedRequest && url && url.trim()) {
    saveCurrentRequest();
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

    const requestData = {
      url: previewUrl || url.trim(),
      method,
      headers: parsedHeaders,
      body: body.trim(),
      params: params.filter(p => p.key && p.key.trim())
    };
    
    console.log('📤 Sending request:', method, requestData.url);
    
    // Save the request when sending
    saveCurrentRequest();
    
    dispatch('submit', requestData);
  }

  // Save current request data
  function saveCurrentRequest() {
    if (!selectedRequest || !url.trim()) return;
    
    let parsedHeaders = {};
    try {
      parsedHeaders = JSON.parse(headers);
    } catch {
      parsedHeaders = {};
    }

    dispatch('save', {
      url: previewUrl || url.trim(),
      method,
      headers: parsedHeaders,
      body: body.trim(),
      params: params.filter(p => p.key && p.key.trim())
    });
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
    
    // Populate form fields
    url = data.url || '';
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
    
    console.log('📥 Loaded request data:', data);
  }

  onMount(() => {
    // Listen for loadRequest events from the parent
    window.addEventListener('loadRequest', loadRequestData);
    
    return () => {
      window.removeEventListener('loadRequest', loadRequestData);
    };
  });

  // Export function so parent can call it when switching requests
  export { saveCurrentRequest };
</script>

<div class="card">
  <h2>🔄 HTTP Request</h2>
  
  <form on:submit|preventDefault={handleSubmit}>
    <div class="form-group">
      <label for="url">URL *</label>
      <input 
        id="url"
        type="text"
        bind:value={url} 
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

    <button type="submit" class="btn btn-primary" disabled={loading || !headersValid || !canSend}>
      {#if loading}
        🔄 Sending...
      {:else if !canSend}
        📤 Select a Request First
      {:else}
        📤 Send Request
      {/if}
    </button>
    
    {#if !canSend && !loading}
      <p class="send-hint">Select a request from the collection to enable sending</p>
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

  .btn-primary {
    width: 100%;
    font-size: 1.1rem;
    padding: 1rem;
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
</style>