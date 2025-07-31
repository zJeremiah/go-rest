<script>
  import { onMount } from 'svelte';
  import RequestForm from '../lib/RequestForm.svelte';
  import ResponseDisplay from '../lib/ResponseDisplay.svelte';

  let response = null;
  let loading = false;
  
  // Saved requests
  let savedRequests = [];
  let selectedRequest = null;
  
  // Variables
  let variables = [];
  let activeCollectionTab = 'requests';
  
  // Resizable sections
  let collectionWidth = 300;
  let requestWidth = 500;
  let isResizing = false;
  let resizeType = '';
  let startX = 0;
  let startCollectionWidth = 0;
  let startRequestWidth = 0;

  async function handleRequest(requestData) {
    loading = true;
    response = null;

    try {
      // Include variables in the request
      const requestWithVariables = {
        ...requestData,
        variables: variables
      };

      console.log('🚀 API request:', requestWithVariables.method, requestWithVariables.url);

      const res = await fetch('/api/proxy', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestWithVariables)
      });

      response = await res.json();
      
      // Update the last response for the current request
      if (response && !response.error && selectedRequest) {
        await updateRequestResponse(selectedRequest.id, response);
      }
    } catch (error) {
      response = {
        error: `Failed to make request: ${error.message}`,
        statusCode: 0,
        status: 'Network Error'
      };
    } finally {
      loading = false;
    }
  }

  async function addNewRequest() {
    const requestName = prompt('Enter a name for the new request:');
    
    if (requestName && requestName.trim()) {
      try {
        const newRequestData = {
          name: requestName.trim(),
          url: 'https://api.example.com/endpoint',
          method: 'GET',
          headers: { 'Content-Type': 'application/json' },
          body: '',
          params: []
        };

        const res = await fetch('/api/requests/save', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(newRequestData)
        });

        if (res.ok) {
          const savedRequest = await res.json();
          savedRequests = [...savedRequests, savedRequest];
          selectRequest(savedRequest);
          console.log('✅ New request created:', savedRequest.name);
        } else {
          console.error('❌ Failed to create request');
        }
      } catch (error) {
        console.error('❌ Error creating request:', error);
      }
    }
  }

  async function updateRequestResponse(requestId, responseData) {
    try {
      const requestToUpdate = savedRequests.find(r => r.id === requestId);
      if (!requestToUpdate) return;

      const updateData = {
        id: requestId,
        name: requestToUpdate.name,
        url: requestToUpdate.url,
        method: requestToUpdate.method,
        headers: requestToUpdate.headers,
        body: requestToUpdate.body,
        params: requestToUpdate.params,
        lastResponse: responseData
      };

      const res = await fetch('/api/requests/update', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updateData)
      });

      if (res.ok) {
        // Update local state
        savedRequests = savedRequests.map(r => 
          r.id === requestId ? { ...r, lastResponse: responseData } : r
        );
        console.log('✅ Request response updated');
      }
    } catch (error) {
      console.error('❌ Error updating request response:', error);
    }
  }

  async function saveRequest(requestData) {
    if (!selectedRequest) return;

    try {
      const updateData = {
        id: selectedRequest.id,
        name: selectedRequest.name,
        url: requestData.url,
        method: requestData.method,
        headers: requestData.headers || {},
        body: requestData.body || '',
        params: requestData.params || []
      };

      const res = await fetch('/api/requests/update', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updateData)
      });

      if (res.ok) {
        // Update local state
        savedRequests = savedRequests.map(r => 
          r.id === selectedRequest.id ? { ...r, ...updateData } : r
        );
        selectedRequest = { ...selectedRequest, ...updateData };
        console.log('✅ Request saved:', selectedRequest.name);
      }
    } catch (error) {
      console.error('❌ Error saving request:', error);
    }
  }

  async function duplicateRequest(request) {
    try {
      const res = await fetch('/api/requests/duplicate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ id: request.id })
      });

      if (res.ok) {
        const duplicatedRequest = await res.json();
        savedRequests = [...savedRequests, duplicatedRequest];
        selectRequest(duplicatedRequest);
        console.log('✅ Request duplicated:', duplicatedRequest.name);
      } else {
        console.error('❌ Failed to duplicate request');
      }
    } catch (error) {
      console.error('❌ Error duplicating request:', error);
    }
  }

  async function deleteRequest(request) {
    if (confirm(`Are you sure you want to delete "${request.name}"?`)) {
      try {
        console.log('🗑️ Deleting request:', request.id, request.name);
        console.log('📊 Requests before delete:', savedRequests.length);
        
        const res = await fetch('/api/requests/delete', {
          method: 'DELETE',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ id: request.id })
        });

        console.log('🔄 Delete response status:', res.status);

        if (res.ok) {
          const responseData = await res.json();
          console.log('✅ Delete response:', responseData);
          
          // Filter out the deleted request
          const newRequests = savedRequests.filter(r => r.id !== request.id);
          console.log('📊 Requests after filter:', newRequests.length);
          
          savedRequests = newRequests;
          
          if (selectedRequest?.id === request.id) {
            selectedRequest = null;
          }
          console.log('✅ Request deleted from UI:', request.name);
        } else {
          const errorData = await res.text();
          console.error('❌ Failed to delete request:', res.status, errorData);
        }
      } catch (error) {
        console.error('❌ Error deleting request:', error);
      }
    }
  }

  async function loadSavedRequests() {
    try {
      console.log('📂 Loading saved requests from server...');
      const res = await fetch('/api/requests');
      if (res.ok) {
        const data = await res.json();
        const newRequests = data.requests || [];
        console.log(`📂 Loaded ${newRequests.length} saved requests from server`);
        console.log('📊 Request IDs:', newRequests.map(r => r.id));
        savedRequests = newRequests;
      }
    } catch (error) {
      console.error('❌ Error loading saved requests:', error);
    }
  }

  async function loadVariables() {
    try {
      const res = await fetch('/api/variables');
      if (res.ok) {
        const data = await res.json();
        variables = data.variables || [];
        console.log(`🔧 Loaded ${variables.length} variables`);
      }
    } catch (error) {
      console.error('❌ Error loading variables:', error);
    }
  }

  async function saveVariables() {
    try {
      const res = await fetch('/api/variables/save', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ variables })
      });

      if (res.ok) {
        console.log('✅ Variables saved');
      } else {
        console.error('❌ Failed to save variables');
      }
    } catch (error) {
      console.error('❌ Error saving variables:', error);
    }
  }

  function addVariable() {
    variables = [...variables, { key: '', value: '' }];
  }

  function removeVariable(index) {
    variables = variables.filter((_, i) => i !== index);
    saveVariables();
  }

  function updateVariable(index, field, value) {
    variables[index][field] = value;
    variables = [...variables]; // Trigger reactivity
    // Debounced save
    clearTimeout(variables.saveTimeout);
    variables.saveTimeout = setTimeout(saveVariables, 1000);
  }

  async function selectRequest(request) {
    // Save current request before switching (if there is one)
    if (selectedRequest && requestFormRef?.saveCurrentRequest) {
      requestFormRef.saveCurrentRequest();
    }
    
    selectedRequest = request;
    // Dispatch custom event to populate the form
    const event = new CustomEvent('loadRequest', {
      detail: {
        url: request.url,
        method: request.method,
        headers: request.headers,
        body: request.body,
        params: request.params || []
      }
    });
    window.dispatchEvent(event);
  }

  // Reference to RequestForm component
  let requestFormRef;

  function startResize(event, type) {
    isResizing = true;
    resizeType = type;
    startX = event.clientX;
    startCollectionWidth = collectionWidth;
    startRequestWidth = requestWidth;
    
    document.addEventListener('mousemove', handleResize);
    document.addEventListener('mouseup', stopResize);
    document.body.style.cursor = 'col-resize';
    event.preventDefault();
  }

  function handleResize(event) {
    if (!isResizing) return;
    
    const deltaX = event.clientX - startX;
    
    if (resizeType === 'collection') {
      const newWidth = Math.max(200, Math.min(500, startCollectionWidth + deltaX));
      collectionWidth = newWidth;
    } else if (resizeType === 'request') {
      const newWidth = Math.max(300, Math.min(1000, startRequestWidth + deltaX));
      requestWidth = newWidth;
    }
  }

  function stopResize() {
    isResizing = false;
    resizeType = '';
    document.removeEventListener('mousemove', handleResize);
    document.removeEventListener('mouseup', stopResize);
    document.body.style.cursor = 'default';
    
    // Save to localStorage
    localStorage.setItem('layoutWidths', JSON.stringify({
      collection: collectionWidth,
      request: requestWidth
    }));
  }

  onMount(() => {
    // Load saved widths from localStorage
    const saved = localStorage.getItem('layoutWidths');
    if (saved) {
      try {
        const { collection, request } = JSON.parse(saved);
        if (collection) collectionWidth = collection;
        if (request) requestWidth = request;
      } catch (e) {
        console.warn('Failed to load saved layout widths');
      }
    }
    
    // Load saved requests and variables from server
    loadSavedRequests();
    loadVariables();
  });
</script>

<div class="container">
  <!-- Collection Section -->
  <div class="collection-section" style="width: {collectionWidth}px;">
    <div class="card">
      <div class="collection-header">
        <h2>📂 Collection</h2>
      </div>
      
      <!-- Collection Tabs -->
      <div class="collection-tabs">
        <button 
          class="collection-tab"
          class:active={activeCollectionTab === 'requests'}
          on:click={() => activeCollectionTab = 'requests'}
        >
          📝 Requests
          {#if savedRequests.length > 0}
            <span class="tab-count">({savedRequests.length})</span>
          {/if}
        </button>
        <button 
          class="collection-tab"
          class:active={activeCollectionTab === 'variables'}
          on:click={() => activeCollectionTab = 'variables'}
        >
          🔧 Variables
          {#if variables.length > 0}
            <span class="tab-count">({variables.length})</span>
          {/if}
        </button>
      </div>

      <!-- Requests Tab Content -->
      {#if activeCollectionTab === 'requests'}
        <div class="tab-content-header">
          <button class="btn-add" on:click={addNewRequest} title="Add new request">
            ➕ Add Request
          </button>
        </div>
        
        {#if savedRequests.length === 0}
          <div class="empty-state">
            <div class="empty-icon">📝</div>
            <p>No saved requests yet.</p>
            <p class="empty-hint">Click "Add Request" to create your first request!</p>
          </div>
        {:else}
          <div class="requests-list">
            {#each savedRequests as request}
              <div 
                class="request-item"
                class:selected={selectedRequest?.id === request.id}
              >
                <div 
                  class="request-content"
                  on:click={() => selectRequest(request)}
                  on:keydown={(e) => e.key === 'Enter' && selectRequest(request)}
                  role="button"
                  tabindex="0"
                  aria-label="Load request: {request.name}"
                >
                  <div class="request-header">
                    <span class="method-badge method-{request.method.toLowerCase()}">{request.method}</span>
                    <span class="request-name">{request.name}</span>
                  </div>
                  <div class="request-url">{request.url}</div>
                  {#if request.lastResponse}
                    <div class="last-response">
                      <span class="status-badge status-{Math.floor(request.lastResponse.statusCode / 100)}xx">
                        {request.lastResponse.statusCode}
                      </span>
                      <span class="response-time">{new Date(request.updatedAt).toLocaleDateString()}</span>
                    </div>
                  {/if}
                </div>
                
                <div class="request-actions">
                  <button 
                    class="action-btn duplicate-btn" 
                    on:click={(e) => { e.stopPropagation(); duplicateRequest(request); }}
                    title="Duplicate request"
                  >
                    📋
                  </button>
                  <button 
                    class="action-btn delete-btn" 
                    on:click={(e) => { e.stopPropagation(); deleteRequest(request); }}
                    title="Delete request"
                  >
                    🗑️
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      {/if}

      <!-- Variables Tab Content -->
      {#if activeCollectionTab === 'variables'}
        <div class="tab-content-header">
          <button class="btn-add" on:click={addVariable} title="Add new variable">
            ➕ Add Variable
          </button>
        </div>
        
        {#if variables.length === 0}
          <div class="empty-state">
            <div class="empty-icon">🔧</div>
            <p>No variables defined yet.</p>
            <p class="empty-hint">Add variables to use in URLs, headers, and params with template syntax.</p>
          </div>
        {:else}
          <div class="variables-list">
            {#each variables as variable, index}
              <div class="variable-item">
                <div class="variable-inputs">
                  <input 
                    type="text" 
                    placeholder="Variable name"
                    bind:value={variable.key}
                    on:input={(e) => updateVariable(index, 'key', e.target.value)}
                    class="variable-key"
                  />
                  <input 
                    type="text" 
                    placeholder="Value"
                    bind:value={variable.value}
                    on:input={(e) => updateVariable(index, 'value', e.target.value)}
                    class="variable-value"
                  />
                </div>
                <button 
                  class="action-btn delete-btn" 
                  on:click={() => removeVariable(index)}
                  title="Delete variable"
                >
                  🗑️
                </button>
                {#if variable.key}
                  <div class="variable-usage">Use: <code>{'{'+'{'}{variable.key}{'}'+'}'}</code></div>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  </div>
  
  <!-- Resize Handle 1 -->
  <button 
    class="resize-handle" 
    class:resizing={isResizing && resizeType === 'collection'}
    on:mousedown={(e) => startResize(e, 'collection')}
    aria-label="Resize collection panel"
  ></button>
  
  <!-- Request Section -->
  <div class="request-section" style="width: {requestWidth}px;">
    <RequestForm 
      bind:this={requestFormRef}
      on:submit={(e) => handleRequest(e.detail)} 
      on:save={(e) => saveRequest(e.detail)}
      {loading} 
      {selectedRequest}
      {variables}
      canSend={!!selectedRequest}
    />
  </div>
  
  <!-- Resize Handle 2 -->
  <button 
    class="resize-handle" 
    class:resizing={isResizing && resizeType === 'request'}
    on:mousedown={(e) => startResize(e, 'request')}
    aria-label="Resize request panel"
  ></button>
  
  <!-- Response Section -->
  <div class="response-section">
    <ResponseDisplay {response} {loading} />
  </div>
</div>

<style>
  .container {
    display: flex;
    width: calc(100vw - 2rem);
    padding: 0 1rem;
    box-sizing: border-box;
    gap: 0;
    user-select: none; /* Prevent text selection during resize */
  }

  /* Mobile: Stack vertically */
  @media (max-width: 1023px) {
    .container {
      flex-direction: column;
      gap: 1.5rem;
    }
    
    .collection-section, 
    .request-section,
    .response-section {
      width: 100% !important;
    }
    
    .resize-handle {
      display: none;
    }
  }

  /* Desktop: Horizontal resizable layout */
  @media (min-width: 1024px) {
    .container {
      flex-direction: row;
      height: calc(100vh - 200px); /* Account for header */
    }
  }

  .collection-section,
  .request-section,
  .response-section {
    min-height: 400px;
    overflow: hidden;
    box-sizing: border-box;
    flex-shrink: 0;
  }

  @media (min-width: 1024px) {
    .response-section {
      flex: 1; /* Takes remaining space */
      min-width: 250px;
    }
  }

  /* Resize Handle Styling */
  .resize-handle {
    width: 6px;
    background: #e5e7eb;
    cursor: col-resize;
    position: relative;
    transition: all 0.2s ease;
    margin: 0 2px;
    border-radius: 3px;
    border: none;
    padding: 0;
  }

  .resize-handle:hover,
  .resize-handle.resizing {
    background: #6366f1;
    box-shadow: 0 0 8px rgba(99, 102, 241, 0.3);
  }

  .resize-handle::before {
    content: '';
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 3px;
    height: 30px;
    background: rgba(255, 255, 255, 0.6);
    border-radius: 1.5px;
    transition: all 0.2s ease;
  }

  .resize-handle:hover::before,
  .resize-handle.resizing::before {
    background: rgba(255, 255, 255, 0.9);
    height: 40px;
  }

  /* Improved resize visual feedback */
  .resize-handle:active {
    background: #4f46e5;
  }

  /* Collection Section Styling */
  .collection-header {
    margin-bottom: 1rem;
  }

  .collection-header h2 {
    margin: 0;
  }

  .collection-tabs {
    display: flex;
    background: #f9fafb;
    border-radius: 6px;
    padding: 4px;
    margin-bottom: 1rem;
    gap: 2px;
  }

  .collection-tab {
    flex: 1;
    background: transparent;
    border: none;
    padding: 0.75rem 1rem;
    border-radius: 4px;
    font-size: 0.875rem;
    font-weight: 500;
    color: #6b7280;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.25rem;
  }

  .collection-tab:hover {
    background: rgba(255, 255, 255, 0.5);
    color: #374151;
  }

  .collection-tab.active {
    background: white;
    color: #3b82f6;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .tab-count {
    font-size: 0.75rem;
    background: #e5e7eb;
    color: #6b7280;
    padding: 0.125rem 0.375rem;
    border-radius: 10px;
    font-weight: 600;
  }

  .collection-tab.active .tab-count {
    background: #dbeafe;
    color: #2563eb;
  }

  .tab-content-header {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 1rem;
  }

  .btn-add {
    background: #3b82f6;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    gap: 0.25rem;
    white-space: nowrap;
  }

  .btn-add:hover {
    background: #2563eb;
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(59, 130, 246, 0.2);
  }

  .btn-add:active {
    transform: translateY(0);
  }

  .empty-state {
    text-align: center;
    padding: 2rem 1rem;
    color: #9ca3af;
  }

  .empty-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
  }

  .empty-hint {
    font-size: 0.875rem;
    margin-top: 0.5rem;
    color: #6b7280;
  }

  .requests-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-height: calc(100vh - 300px);
    overflow-y: auto;
  }

  .request-item {
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    transition: all 0.2s ease;
    display: flex;
    align-items: stretch;
    overflow: hidden;
  }

  .request-item:hover {
    background: #f3f4f6;
    border-color: #d1d5db;
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }

  .request-item.selected {
    background: #eff6ff;
    border-color: #3b82f6;
    box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
  }

  .request-content {
    flex: 1;
    padding: 0.75rem;
    cursor: pointer;
  }

  .request-actions {
    display: flex;
    flex-direction: column;
    background: rgba(255, 255, 255, 0.5);
    border-left: 1px solid #e5e7eb;
  }

  .action-btn {
    background: transparent;
    border: none;
    padding: 0.5rem;
    cursor: pointer;
    font-size: 0.875rem;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 32px;
    min-width: 32px;
  }

  .action-btn:hover {
    background: rgba(255, 255, 255, 0.8);
  }

  .duplicate-btn:hover {
    background: #fef3c7;
    color: #d97706;
  }

  .delete-btn {
    border-top: 1px solid #e5e7eb;
  }

  .delete-btn:hover {
    background: #fee2e2;
    color: #dc2626;
  }

  .request-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .method-badge {
    padding: 0.125rem 0.375rem;
    border-radius: 0.25rem;
    font-size: 0.625rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .method-get { background: #dcfce7; color: #16a34a; }
  .method-post { background: #fef3c7; color: #d97706; }
  .method-put { background: #dbeafe; color: #2563eb; }
  .method-delete { background: #fee2e2; color: #dc2626; }
  .method-patch { background: #f3e8ff; color: #9333ea; }
  .method-head, .method-options { background: #f1f5f9; color: #64748b; }

  .request-name {
    font-weight: 500;
    color: #374151;
    font-size: 0.875rem;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .request-url {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.75rem;
    color: #6b7280;
    margin-bottom: 0.5rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .last-response {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }

  .status-badge {
    padding: 0.125rem 0.375rem;
    border-radius: 0.25rem;
    font-size: 0.625rem;
    font-weight: 600;
  }

  .status-2xx { background: #dcfce7; color: #16a34a; }
  .status-3xx { background: #fef3c7; color: #d97706; }
  .status-4xx { background: #fee2e2; color: #dc2626; }
  .status-5xx { background: #fef2f2; color: #b91c1c; }

  .response-time {
    font-size: 0.625rem;
    color: #9ca3af;
  }

  /* Variables Section Styling */
  .variables-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    max-height: calc(100vh - 400px);
    overflow-y: auto;
  }

  .variable-item {
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    padding: 0.75rem;
    transition: all 0.2s ease;
  }

  .variable-item:hover {
    background: #f3f4f6;
    border-color: #d1d5db;
  }

  .variable-inputs {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .variable-key,
  .variable-value {
    flex: 1;
    padding: 0.5rem;
    border: 1px solid #d1d5db;
    border-radius: 4px;
    font-size: 0.875rem;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    transition: all 0.2s ease;
  }

  .variable-key:focus,
  .variable-value:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
  }

  .variable-usage {
    font-size: 0.75rem;
    color: #6b7280;
    margin-top: 0.5rem;
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .variable-usage code {
    background: #f3f4f6;
    color: #1f2937;
    padding: 0.125rem 0.375rem;
    border-radius: 3px;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .variable-item {
    position: relative;
  }

  .variable-item .action-btn {
    position: absolute;
    top: 0.75rem;
    right: 0.75rem;
    margin: 0;
  }

  /* Smooth transitions for resize */
  .collection-section,
  .request-section {
    transition: width 0.1s ease-out;
  }
</style>