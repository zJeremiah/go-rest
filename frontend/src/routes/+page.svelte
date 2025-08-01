<svelte:head>
  <script>
    // Prevent FOUC by loading theme immediately
    (function() {
      if (typeof localStorage !== 'undefined') {
        const savedTheme = localStorage.getItem('selectedTheme');
        if (savedTheme) {
          document.documentElement.setAttribute('data-theme', savedTheme);
        }
      }
    })();
  </script>
</svelte:head>

<script>
  import { onMount } from 'svelte';
  import RequestForm from '../lib/RequestForm.svelte';
  import ResponseDisplay from '../lib/ResponseDisplay.svelte';

  let response = null;
  let loading = false;
  
  // Saved requests
  let savedRequests = [];
  let selectedRequest = null;
  let renamingRequestId = null;
  let newRequestName = '';
  
  // Variables
  let variables = [];
  let activeCollectionTab = 'requests';
  
  // Theme system
  let currentTheme = 'light';
  const themes = [
    { id: 'light', name: '☀️ Light', label: 'Light Theme' },
    { id: 'dark', name: '🌙 Dark', label: 'Dark Theme' },
    { id: 'blue', name: '💙 Blue', label: 'Blue Theme' },
    { id: 'green', name: '💚 Green', label: 'Green Theme' }
  ];
  
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
          selectRequest(savedRequest); // This will automatically save to localStorage
          console.log('✅ New request created and selected:', savedRequest.name);
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
    console.log('🎯 Parent saveRequest function called');
    console.log('🔍 Received requestData:', requestData);
    console.log('🔍 selectedRequest:', selectedRequest);
    
    if (!selectedRequest) {
      console.log('❌ No selected request, aborting save');
      return;
    }

    console.log('🔄 Parent received save request for:', selectedRequest.name);
    console.log('🔍 NEW URL from form:', requestData.url);
    console.log('🔍 OLD URL in state:', selectedRequest.url);
    console.log('🔍 URLs are different:', requestData.url !== selectedRequest.url);
    console.log('🔍 Will update URL:', selectedRequest.url, '→', requestData.url);

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

      console.log('📤 Sending update to backend:', updateData);
      console.log('🌐 Making API call to /api/requests/update');

      const res = await fetch('/api/requests/update', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updateData)
      });

      console.log('📨 API response status:', res.status);
      
      if (res.ok) {
        console.log('✅ API call successful');

        // Update local state with proper isolation
        const oldUrl = selectedRequest.url;

        // Create a properly isolated updated request object
        const updatedRequest = { ...selectedRequest, ...updateData };

        // Update savedRequests array with proper isolation
        savedRequests = savedRequests.map(r => 
          r.id === selectedRequest.id ? updatedRequest : r
        );

        // Update selectedRequest reference
        selectedRequest = updatedRequest;

        console.log('✅ Request auto-saved:', selectedRequest.name);
        console.log('🔄 URL updated in local state from:', oldUrl, 'to:', updateData.url);
        console.log('📊 Updated  savedRequests array length:', savedRequests.length);

        // Optional: Could dispatch custom event to notify child component save is complete
        // This helps with the save status feedback
      } else {
        const errorText = await res.text();
        console.error('❌ Failed to save request:', res.status, errorText);
        console.error('📄 Error response body:', errorText);

        // Show user-friendly error message for file locking issues
        if (errorText.includes('file may be locked') || errorText.includes('Access is denied')) {
          console.warn('⚠️  File temporarily locked - save will be retried automatically');
        }
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
        selectRequest(duplicatedRequest); // This will automatically save to localStorage
        console.log('✅ Request duplicated and selected:', duplicatedRequest.name);
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
            // Clear localStorage since the selected request was deleted
            localStorage.removeItem('lastSelectedRequestId');
            console.log('🗑️  Cleared last selected request from localStorage');
    
            // Auto-select another request if available
            if (newRequests.length > 0) {
              console.log('🔄 Auto-selecting another request after deletion');
              selectRequest(newRequests[0]);
            }
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

  async function startRenameRequest(request) {
    console.log('🏷️ Starting rename for request:', request.id, request.name);
    renamingRequestId = request.id;
    newRequestName = request.name || '';
    console.log('🏷️ Set renamingRequestId:', renamingRequestId, 'newRequestName:', newRequestName);
  }

  function cancelRename() {
    renamingRequestId = null;
    newRequestName = '';
  }

  async function saveRename(request) {
    console.log('💾 Saving rename for request:', request.id, 'new name:', newRequestName);
    
    if (!newRequestName || !newRequestName.trim()) {
      console.log('❌ Empty name detected, canceling rename');
      cancelRename();
      return;
    }

    try {
      const updateData = {
        id: request.id,
        name: newRequestName.trim(),
        url: request.url,
        method: request.method,
        headers: request.headers,
        body: request.body,
        params: request.params
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
        const updatedRequest = { ...request, name: newRequestName.trim() };

        savedRequests = savedRequests.map(r => 
          r.id === request.id ? updatedRequest : r
        );

        // Update selectedRequest if it's the one being renamed
        if (selectedRequest?.id === request.id) {
          selectedRequest = updatedRequest;
        }

        console.log('✅ Request renamed:', newRequestName.trim());
        cancelRename();
      } else {
        console.error('❌ Failed to rename request');
        alert('Failed to rename request. Please try again.');
      }
    } catch (error) {
      console.error('❌ Error renaming request:', error);
      alert('Failed to rename request. Please try again.');
    }
  }

  function handleRenameKeydown(event, request) {
    if (event.key === 'Enter') {
      saveRename(request);
    } else if (event.key === 'Escape') {
      cancelRename();
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

        // Ensure proper isolation by creating new objects
        savedRequests = newRequests.map(req => ({
          ...req,
          id: req.id,
          name: req.name,
          url: req.url,
          method: req.method,
          headers: req.headers ? { ...req.headers } : {},
          body: req.body || '',
          params: req.params ? [...req.params] : [],
          lastResponse: req.lastResponse ? { ...req.lastResponse } : null,
          createdAt: req.createdAt,
          updatedAt: req.updatedAt
        }));

        // Auto-select the last selected request after loading
        autoSelectLastRequest();
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
    console.log('🎯 SELECTING:', request.name, 'URL:', request.url);
    
    // Don't auto-save when switching requests - this causes data corruption
    // Users should manually save changes if needed
    
    selectedRequest = request;
    
    // Store selected request ID in localStorage for persistence
    localStorage.setItem('lastSelectedRequestId', request.id);
    
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

  // Auto-select the last selected request
  function autoSelectLastRequest() {
    const lastSelectedId = localStorage.getItem('lastSelectedRequestId');
    if (lastSelectedId && savedRequests.length > 0) {
      console.log('🔄 Looking for last selected request ID:', lastSelectedId);
      
      const lastRequest = savedRequests.find(r => r.id === lastSelectedId);
      if (lastRequest) {
        console.log('✅ Auto-selecting last request:', lastRequest.name);
        selectRequest(lastRequest);
      } else {
        console.log('⚠️  Last selected request not found, selecting first available');
        // If the last selected request doesn't exist anymore, select the first one
        if (savedRequests.length > 0) {
          selectRequest(savedRequests[0]);
        }
      }
    } else if (savedRequests.length > 0) {
      console.log('📝 No previous selection, selecting first request');
      // If no previous selection, select the first request
      selectRequest(savedRequests[0]);
    }
  }

  // Reference to RequestForm component
  let requestFormRef;

  // Accessible focus action for rename input
  function focusOnMount(element) {
    element.focus();
    element.select();
  }

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

  // Theme functions
  function applyTheme(theme) {
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', theme);
    }
    currentTheme = theme;
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('selectedTheme', theme);
    }
  }

  function loadSavedTheme() {
    if (typeof localStorage !== 'undefined') {
      const savedTheme = localStorage.getItem('selectedTheme');
      if (savedTheme && themes.find(t => t.id === savedTheme)) {
        applyTheme(savedTheme);
      }
    }
  }

  // Load theme immediately to prevent FOUC
  loadSavedTheme();

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
        <div class="theme-selector">
          <label for="theme-select">Theme:</label>
          <select 
            id="theme-select" 
            bind:value={currentTheme} 
            on:change={(e) => applyTheme(e.target.value)}
            class="theme-select"
          >
            {#each themes as theme}
              <option value={theme.id}>{theme.name}</option>
            {/each}
          </select>
        </div>
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
                    {#if renamingRequestId === request.id}
                      <input 
                        type="text" 
                        bind:value={newRequestName}
                        on:keydown={(e) => handleRenameKeydown(e, request)}
                        on:blur={() => saveRename(request)}
                        on:click={(e) => e.stopPropagation()}
                        class="rename-input"
                        use:focusOnMount
                      />
                    {:else}
                      <span class="request-name">{request.name}</span>
                    {/if}
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
                    class="action-btn rename-btn" 
                    on:click={(e) => { e.stopPropagation(); startRenameRequest(request); }}
                    title="Rename request"
                  >
                    ✏️
                  </button>
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
                <div class="variable-footer">
                  {#if variable.key}
                    <div class="variable-usage">Use: <code>{'{'+'{'}{variable.key}{'}'+'}'}</code></div>
                  {/if}
                  <button 
                    class="variable-delete-btn" 
                    on:click={() => removeVariable(index)}
                    title="Delete variable"
                  >
                    🗑️ Delete
                  </button>
                </div>
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
  /* Theme System - CSS Custom Properties */
  :global(:root) {
    /* Light Theme (Default) */
    --bg-primary: #ffffff;
    --bg-secondary: #f9fafb;
    --bg-tertiary: #f3f4f6;
    --bg-accent: #eff6ff;
    --tabs-container-bg: #f9fafb;
    --request-item-bg: #f9fafb;
    --text-primary: #374151;
    --text-secondary: #6b7280;
    --text-accent: #3b82f6;
    --border-primary: #e5e7eb;
    --border-secondary: #d1d5db;
    --border-accent: #3b82f6;
    --button-primary: #3b82f6;
    --button-primary-hover: #2563eb;
    --button-secondary: #f3f4f6;
    --button-secondary-hover: #e5e7eb;
    --success: #16a34a;
    --warning: #d97706;
    --error: #dc2626;
    --preview-bg: #f0f9ff;
    --preview-border: #bae6fd;
    --preview-text: #0369a1;
  }

  :global([data-theme="dark"]) {
    --bg-primary: #1f2937;
    --bg-secondary: #454545;
    --bg-tertiary: #393939;
    --bg-accent: #1e3a8a;
    --tabs-container-bg: #374151;
    --request-item-bg: #2d3748;
    --text-primary: #f9fafb;
    --text-secondary: #d1d5db;
    --text-accent: #60a5fa;
    --border-primary: #374151;
    --border-secondary: #4b5563;
    --border-accent: #60a5fa;
    --button-primary: #3b82f6;
    --button-primary-hover: #2563eb;
    --button-secondary: #374151;
    --button-secondary-hover: #4b5563;
    --success: #22c55e;
    --warning: #f59e0b;
    --error: #ef4444;
    --preview-bg: #344a86;
    --preview-border: #3b82f6;
    --preview-text: #93c5fd;
  }

  :global([data-theme="blue"]) {
    --bg-primary: #f0f9ff;
    --bg-secondary: #e0f2fe;
    --bg-tertiary: #bae6fd;
    --bg-accent: #dbeafe;
    --tabs-container-bg: #e0f2fe;
    --request-item-bg: #f0f9ff;
    --text-primary: #0c4a6e;
    --text-secondary: #0369a1;
    --text-accent: #2563eb;
    --border-primary: #7dd3fc;
    --border-secondary: #38bdf8;
    --border-accent: #2563eb;
    --button-primary: #0ea5e9;
    --button-primary-hover: #0284c7;
    --button-secondary: #e0f2fe;
    --button-secondary-hover: #bae6fd;
    --success: #059669;
    --warning: #d97706;
    --error: #dc2626;
    --preview-bg: #dbeafe;
    --preview-border: #93c5fd;
    --preview-text: #1e40af;
  }

  :global([data-theme="green"]) {
    --bg-primary: #f0fdf4;
    --bg-secondary: #dcfce7;
    --bg-tertiary: #bbf7d0;
    --bg-accent: #d1fae5;
    --tabs-container-bg: #dcfce7;
    --request-item-bg: #f0fdf4;
    --text-primary: #14532d;
    --text-secondary: #166534;
    --text-accent: #059669;
    --border-primary: #86efac;
    --border-secondary: #4ade80;
    --border-accent: #059669;
    --button-primary: #10b981;
    --button-primary-hover: #059669;
    --button-secondary: #dcfce7;
    --button-secondary-hover: #bbf7d0;
    --success: #16a34a;
    --warning: #d97706;
    --error: #dc2626;
    --preview-bg: #d1fae5;
    --preview-border: #86efac;
    --preview-text: #047857;
  }

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
      min-height: calc(100vh - 200px); /* Account for header, but allow expansion */
      max-height: calc(100vh - 140px); /* More flexible max height */
    }
  }

  .collection-section,
  .request-section,
  .response-section {
    min-height: 400px;
    overflow-y: auto;
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
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    gap: 1rem;
  }

  .collection-header h2 {
    margin: 0;
    color: var(--text-primary);
  }

  /* Theme Selector */
  .theme-selector {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .theme-selector label {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .theme-select {
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-primary);
    border-radius: 4px;
    padding: 0.375rem 0.75rem;
    font-size: 0.875rem;
    cursor: pointer;
    transition: all 0.2s ease;
    min-width: 120px;
  }

  .theme-select:hover {
    border-color: var(--border-accent);
  }

  .theme-select:focus {
    outline: none;
    border-color: var(--border-accent);
    box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
  }

  .collection-tabs {
    display: flex;
    background: var(--tabs-container-bg);
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
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.25rem;
  }

  .collection-tab:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }

  .collection-tab.active {
    background: var(--bg-primary);
    color: var(--text-accent);
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
    background: var(--button-primary);
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
    background: var(--button-primary-hover);
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
    background: var(--request-item-bg);
    border: 1px solid var(--border-primary);
    border-radius: 6px;
    transition: all 0.2s ease;
    display: flex;
    align-items: stretch;
    overflow: hidden;
  }

  .request-item:hover {
    background: var(--bg-tertiary);
    border-color: var(--border-secondary);
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }

  .request-item.selected {
    background: var(--bg-accent);
    border-color: var(--border-accent);
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

  .rename-btn:hover {
    background: #fef3c7;
    color: #d97706;
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

  .rename-input {
    flex: 1;
    font-weight: 500;
    color: #374151;
    font-size: 0.875rem;
    border: 1px solid #3b82f6;
    border-radius: 4px;
    padding: 0.25rem 0.5rem;
    background: white;
    outline: none;
    box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
  }

  .rename-input:focus {
    border-color: #2563eb;
    box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.2);
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
    display: flex;
    align-items: center;
    gap: 0.25rem;
    flex: 1;
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

  .variable-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 0.75rem;
    gap: 1rem;
  }

  .variable-delete-btn {
    background: #fef2f2;
    color: #dc2626;
    border: 1px solid #fecaca;
    padding: 0.375rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
    font-weight: 500;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    gap: 0.25rem;
    white-space: nowrap;
  }

  .variable-delete-btn:hover {
    background: #fee2e2;
    border-color: #f87171;
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(220, 38, 38, 0.15);
  }

  .variable-delete-btn:active {
    transform: translateY(0);
  }

  /* Smooth transitions for resize */
  .collection-section,
  .request-section {
    transition: width 0.1s ease-out;
  }
</style>