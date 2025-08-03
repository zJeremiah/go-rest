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
  
  // Variables and Environments
  let variables = [];
  let environments = [];
  let currentEnvironment = null;
  let activeCollectionTab = 'requests';
  
  // Groups
  let groups = [];
  let selectedGroup = 'all'; // Start with 'all' to show everything initially
  let filteredRequests = [];
  
  // UI Settings
  let wordWrap = false;
  
  // Environment modal states
  let showCreateEnvironmentModal = false;
  let showCopyEnvironmentModal = false;
  let newEnvironmentName = '';
  let copySourceEnvironmentId = '';
  let copyTargetEnvironmentId = '';
  
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
        // Determine the group for the new request
        let targetGroup = 'default';
        if (selectedGroup !== 'all') {
          targetGroup = selectedGroup;
        } else if (groups.length > 0) {
          // If 'all' is selected, use the first available group (usually 'default')
          targetGroup = groups.find(g => g.name === 'default')?.name || groups[0].name;
        }

        const newRequestData = {
          name: requestName.trim(),
          url: 'https://api.example.com/endpoint',
          method: 'GET',
          headers: { 'Content-Type': 'application/json' },
          body: '',
          params: [],
          group: targetGroup,
          description: ''
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
        group: requestToUpdate.group || 'default',
        description: requestToUpdate.description || '',
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

      }
    } catch (error) {
      console.error('❌ Error updating request response:', error);
    }
  }

  async function saveRequest(requestData) {

    
    if (!selectedRequest) {

      return;
    }



    try {
      const updateData = {
        id: selectedRequest.id,
        name: selectedRequest.name,
        url: requestData.url,
        method: requestData.method,
        headers: requestData.headers || {},
        body: requestData.body || '',
        params: requestData.params || [],
        group: requestData.group || selectedRequest.group || 'default',
        description: requestData.description || ''
      };





      const res = await fetch('/api/requests/update', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updateData)
      });


      
      if (res.ok) {


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
        

        const res = await fetch('/api/requests/delete', {
          method: 'DELETE',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ id: request.id })
        });

  

        if (res.ok) {
          const responseData = await res.json();
  
  
          // Filter out the deleted request
          const newRequests = savedRequests.filter(r => r.id !== request.id);
    
  
          savedRequests = newRequests;
  
            if (selectedRequest?.id === request.id) {
            selectedRequest = null;
            // Clear localStorage since the selected request was deleted
            localStorage.removeItem('lastSelectedRequestId');
    
    
            // Auto-select another request if available
            if (newRequests.length > 0) {
      
              selectRequest(newRequests[0]);
            }
          }
    
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
    renamingRequestId = request.id;
    newRequestName = request.name || '';
  }

  function cancelRename() {
    renamingRequestId = null;
    newRequestName = '';
  }

  async function saveRename(request) {

    
    if (!newRequestName || !newRequestName.trim()) {

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
        params: request.params,
        group: request.group || 'default'
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

      const res = await fetch('/api/requests');
      if (res.ok) {
        const data = await res.json();
        const newRequests = data.requests || [];
        
        // Load word wrap setting
        wordWrap = data.wordWrap || false;

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
        
        // Force filtering after requests are loaded
        if (groups.length > 0) {
          filterRequestsByGroup();
        }
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

      } else {
        console.error('❌ Failed to save variables');
      }
    } catch (error) {
      console.error('❌ Error saving variables:', error);
    }
  }

  async function saveWordWrap() {
    try {
      const res = await fetch('/api/settings/wordwrap', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ wordWrap })
      });

      if (!res.ok) {
        console.error('❌ Failed to save word wrap setting');
      }
    } catch (error) {
      console.error('❌ Error saving word wrap setting:', error);
    }
  }

  // Handle word wrap change from ResponseDisplay component
  function handleWordWrapChange(event) {
    wordWrap = event.detail.wordWrap;
    saveWordWrap();
  }

  // Environment management functions
  async function loadEnvironments() {
    try {
      const res = await fetch('/api/environments');
      if (res.ok) {
        const data = await res.json();
        environments = data.environments || [];
        currentEnvironment = data.currentEnvironment;

        
        // Load variables from current environment
        await loadVariables();
      }
    } catch (error) {
      console.error('❌ Error loading environments:', error);
    }
  }

  async function createEnvironment(name) {
    try {
      const res = await fetch('/api/environments', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name })
      });

      if (res.ok) {
        const newEnv = await res.json();

        await loadEnvironments(); // Refresh environments
        return newEnv;
      } else {
        const error = await res.text();
        console.error('❌ Failed to create environment:', error);
        throw new Error(error);
      }
    } catch (error) {
      console.error('❌ Error creating environment:', error);
      throw error;
    }
  }

  async function deleteEnvironment(envId) {
    try {
      const res = await fetch(`/api/environments/${envId}`, {
        method: 'DELETE'
      });

      if (res.ok) {

        await loadEnvironments(); // Refresh environments
      } else {
        const error = await res.text();
        console.error('❌ Failed to delete environment:', error);
        throw new Error(error);
      }
    } catch (error) {
      console.error('❌ Error deleting environment:', error);
      throw error;
    }
  }

  async function activateEnvironment(envId) {
    try {
      const res = await fetch(`/api/environments/${envId}/activate`, {
        method: 'POST'
      });

      if (res.ok) {

        currentEnvironment = envId;
        await loadVariables(); // Reload variables for new environment
      } else {
        const error = await res.text();
        console.error('❌ Failed to activate environment:', error);
        throw new Error(error);
      }
    } catch (error) {
      console.error('❌ Error activating environment:', error);
      throw error;
    }
  }

  async function copyEnvironmentVariables(sourceEnvId, targetEnvId) {
    try {
      const res = await fetch(`/api/environments/${targetEnvId}/copy`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ sourceEnvironmentId: sourceEnvId })
      });

      if (res.ok) {

        if (targetEnvId === currentEnvironment) {
          await loadVariables(); // Reload if we're copying to current environment
        }
      } else {
        const error = await res.text();
        console.error('❌ Failed to copy variables:', error);
        throw new Error(error);
      }
    } catch (error) {
      console.error('❌ Error copying variables:', error);
      throw error;
    }
  }

  // Group management functions
  async function loadGroups() {
    try {
      const res = await fetch('/api/groups');
      if (res.ok) {
        const data = await res.json();
        groups = data.groups || [];

        // Auto-select the last selected group after loading
        autoSelectLastGroup();
        
        // Force filtering after groups are loaded
        filterRequestsByGroup();
      }
    } catch (error) {
      console.error('❌ Error loading groups:', error);
    }
  }

  async function createGroup(name) {
    try {
      const res = await fetch('/api/groups', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name })
      });

      if (res.ok) {
        const newGroup = await res.json();

        await loadGroups(); // Refresh groups
        return newGroup;
      } else {
        const error = await res.text();
        console.error('❌ Failed to create group:', error);
        throw new Error(error);
      }
    } catch (error) {
      console.error('❌ Error creating group:', error);
      throw error;
    }
  }

  async function deleteGroup(groupId) {
    try {
      const res = await fetch(`/api/groups/${groupId}`, {
        method: 'DELETE'
      });

      if (res.ok) {

        await loadGroups(); // Refresh groups
        await loadSavedRequests(); // Refresh requests in case any moved
      } else {
        const error = await res.text();
        console.error('❌ Failed to delete group:', error);
        throw new Error(error);
      }
    } catch (error) {
      console.error('❌ Error deleting group:', error);
      throw error;
    }
  }

  async function handleCreateGroup() {
    const groupName = prompt('Enter a name for the new group:');
    if (!groupName || !groupName.trim()) {
      return;
    }

    try {
      await createGroup(groupName.trim());
    } catch (error) {
      alert('Failed to create group: ' + error.message);
    }
  }

  async function handleDeleteGroup(group) {
    if (group.name === 'default') {
      alert('Cannot delete the default group');
      return;
    }

    const hasRequests = savedRequests.some(req => req.group === group.name);
    if (hasRequests) {
      alert('Cannot delete a group that contains requests. Move or delete all requests in this group first.');
      return;
    }

    if (!confirm(`Delete group "${group.name}"?`)) {
      return;
    }

    try {
      await deleteGroup(group.id);
    } catch (error) {
      alert('Failed to delete group: ' + error.message);
    }
  }

  // Filter requests by selected group
  function filterRequestsByGroup() {
    if (selectedGroup === 'all') {
      filteredRequests = [...savedRequests];
    } else {
      filteredRequests = savedRequests.filter(req => req.group === selectedGroup);
    }
    
    // Store selected group in localStorage for persistence
    localStorage.setItem('lastSelectedGroup', selectedGroup);
  }

  // Reactive statement to update filtered requests when savedRequests or selectedGroup changes
  $: if (savedRequests && savedRequests.length >= 0 && groups.length > 0) {
    filterRequestsByGroup();
  }

  // Environment modal handlers
  async function handleCreateEnvironment() {
    if (!newEnvironmentName.trim()) {
      alert('Please enter an environment name');
      return;
    }

    try {
      await createEnvironment(newEnvironmentName.trim());
      newEnvironmentName = '';
      showCreateEnvironmentModal = false;
    } catch (error) {
      alert('Failed to create environment: ' + error.message);
    }
  }

  async function handleCopyEnvironmentVariables() {
    if (!copySourceEnvironmentId || !copyTargetEnvironmentId) {
      alert('Please select both source and target environments');
      return;
    }

    if (copySourceEnvironmentId === copyTargetEnvironmentId) {
      alert('Source and target environments must be different');
      return;
    }

    try {
      await copyEnvironmentVariables(copySourceEnvironmentId, copyTargetEnvironmentId);
      copySourceEnvironmentId = '';
      copyTargetEnvironmentId = '';
      showCopyEnvironmentModal = false;
      alert('Variables copied successfully!');
    } catch (error) {
      alert('Failed to copy variables: ' + error.message);
    }
  }

  async function handleDeleteCurrentEnvironment() {
    if (environments.length <= 1) {
      alert('Cannot delete the last environment');
      return;
    }

    const currentEnv = environments.find(env => env.id === currentEnvironment);
    if (!currentEnv) {
      alert('Current environment not found');
      return;
    }

    if (confirm(`Are you sure you want to delete the "${currentEnv.name}" environment? This action cannot be undone.`)) {
      try {
        await deleteEnvironment(currentEnvironment);
      } catch (error) {
        alert('Failed to delete environment: ' + error.message);
      }
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

    
    // Don't auto-save when switching requests - this causes data corruption
    // Users should manually save changes if needed
    
    selectedRequest = request;
    
    // Store selected request ID in localStorage for persistence
    localStorage.setItem('lastSelectedRequestId', request.id);
    
    // Load the last response if it exists
    if (request.lastResponse) {
      response = request.lastResponse;

    } else {
      response = null;

    }
    
    // Dispatch custom event to populate the form
    const event = new CustomEvent('loadRequest', {
      detail: {
        url: request.url,
        method: request.method,
        headers: request.headers,
        body: request.body,
        bodyType: request.bodyType,
        bodyText: request.bodyText,
        bodyJson: request.bodyJson,
        bodyForm: request.bodyForm,
        params: request.params || [],
        description: request.description || ''
      }
    });
    window.dispatchEvent(event);
  }

  // Auto-select the last selected request
  function autoSelectLastRequest() {
    const lastSelectedId = localStorage.getItem('lastSelectedRequestId');
    if (lastSelectedId && savedRequests.length > 0) {

      
      const lastRequest = savedRequests.find(r => r.id === lastSelectedId);
      if (lastRequest) {

        selectRequest(lastRequest);
      } else {

        // If the last selected request doesn't exist anymore, select the first one
        if (savedRequests.length > 0) {
          selectRequest(savedRequests[0]);
        }
      }
    } else if (savedRequests.length > 0) {

      // If no previous selection, select the first request
      selectRequest(savedRequests[0]);
    }
  }

  // Auto-select the last selected group
  function autoSelectLastGroup() {
    const lastSelectedGroup = localStorage.getItem('lastSelectedGroup');
    if (lastSelectedGroup && groups.length > 0) {
      // Check if the last selected group still exists (including 'all')
      if (lastSelectedGroup === 'all' || groups.some(g => g.name === lastSelectedGroup)) {
        selectedGroup = lastSelectedGroup;
      } else {
        // If the last selected group doesn't exist anymore, default to 'all'
        selectedGroup = 'all';
      }
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
    
    // Load saved requests and environments from server
    loadSavedRequests();
    loadEnvironments();
    loadGroups();
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
          <button class="btn-add" on:click={handleCreateGroup} title="Create new group">
            📁 New Group
          </button>
        </div>

        <!-- Environment Management Section -->
        <div class="environment-section">
          <div class="environment-header">
            <h3>🌍 Environment</h3>
            <div class="environment-actions">
              <button 
                class="btn-env btn-create" 
                on:click={() => showCreateEnvironmentModal = true}
                title="Create new environment"
              >
                ➕ New
              </button>
              {#if environments.length > 1}
                <button 
                  class="btn-env btn-copy" 
                  on:click={() => showCopyEnvironmentModal = true}
                  title="Copy variables between environments"
                >
                  📋 Copy
                </button>
                <button 
                  class="btn-env btn-delete" 
                  on:click={() => handleDeleteCurrentEnvironment()}
                  title="Delete current environment"
                >
                  🗑️ Delete
                </button>
              {/if}
            </div>
          </div>
          
          <div class="environment-selector">
            <label for="env-select">Active Environment:</label>
            <select 
              id="env-select"
              bind:value={currentEnvironment}
              on:change={(e) => activateEnvironment(e.target.value)}
            >
              {#each environments as env}
                <option value={env.id}>{env.name}</option>
              {/each}
            </select>
          </div>
        </div>

        <!-- Group Filter Section -->
        <div class="group-filter-section">
          <div class="group-selector">
            <label for="group-select">Filter by Group:</label>
            <select 
              id="group-select" 
              bind:value={selectedGroup} 
              on:change={() => filterRequestsByGroup()}
              class="group-select"
            >
              <option value="all">All Groups</option>
              {#each groups as group}
                <option value={group.name}>{group.name}</option>
              {/each}
            </select>
          </div>
          <div class="group-actions">
            {#if selectedGroup !== 'all' && selectedGroup !== 'default'}
              <button 
                class="btn-group-delete" 
                on:click={() => handleDeleteGroup(groups.find(g => g.name === selectedGroup))}
                title="Delete selected group"
              >
                🗑️ Delete Group
              </button>
            {/if}
          </div>
        </div>

        {#if savedRequests.length === 0}
          <div class="empty-state">
            <div class="empty-icon">📝</div>
            <p>No saved requests yet.</p>
            <p class="empty-hint">Click "Add Request" to create your first request!</p>
          </div>
        {:else if filteredRequests.length === 0}
          <div class="empty-state">
            <div class="empty-icon">📁</div>
            <p>No requests in "{selectedGroup}" group.</p>
            <p class="empty-hint">Add a request to this group or select a different group.</p>
          </div>
        {:else}
          <div class="requests-list">
            {#each filteredRequests as request}
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
                  {#if request.description && request.description.trim()}
                    <div 
                      class="request-description"
                      title={request.description}
                    >
                      {request.description.split('\n')[0]}
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
        <!-- Variables Section -->
        <div class="variables-section">
          <div class="variables-header">
            <h3>🔧 Variables</h3>
            <button class="btn-add" on:click={addVariable} title="Add new variable">
              ➕ Add Variable
            </button>
          </div>
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
      {groups}
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
    <ResponseDisplay {response} {loading} {wordWrap} on:wordWrapChange={handleWordWrapChange} />
  </div>
</div>

<!-- Environment Management Modals -->
{#if showCreateEnvironmentModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div class="modal-overlay" on:click={() => showCreateEnvironmentModal = false} on:keydown={(e) => e.key === 'Escape' && (showCreateEnvironmentModal = false)} role="button" tabindex="-1">
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
      <div class="modal-header">
        <h3>🌍 Create New Environment</h3>
        <button class="modal-close" on:click={() => showCreateEnvironmentModal = false}>✕</button>
      </div>
      <div class="modal-body">
        <label for="new-env-name">Environment Name:</label>
        <input 
          id="new-env-name"
          type="text" 
          bind:value={newEnvironmentName}
          placeholder="e.g., Development, Staging, Production"
          on:keydown={(e) => e.key === 'Enter' && handleCreateEnvironment()}
        />
      </div>
      <div class="modal-footer">
        <button class="btn-secondary" on:click={() => showCreateEnvironmentModal = false}>Cancel</button>
        <button class="btn-primary" on:click={handleCreateEnvironment}>Create</button>
      </div>
    </div>
  </div>
{/if}

{#if showCopyEnvironmentModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div class="modal-overlay" on:click={() => showCopyEnvironmentModal = false} on:keydown={(e) => e.key === 'Escape' && (showCopyEnvironmentModal = false)} role="button" tabindex="-1">
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
      <div class="modal-header">
        <h3>📋 Copy Variables Between Environments</h3>
        <button class="modal-close" on:click={() => showCopyEnvironmentModal = false}>✕</button>
      </div>
      <div class="modal-body">
        <div class="copy-form">
          <div class="form-group">
            <label for="copy-source">From (Source):</label>
            <select id="copy-source" bind:value={copySourceEnvironmentId}>
              <option value="">Select source environment</option>
              {#each environments as env}
                <option value={env.id}>{env.name}</option>
              {/each}
            </select>
          </div>
          <div class="copy-arrow">→</div>
          <div class="form-group">
            <label for="copy-target">To (Target):</label>
            <select id="copy-target" bind:value={copyTargetEnvironmentId}>
              <option value="">Select target environment</option>
              {#each environments as env}
                <option value={env.id}>{env.name}</option>
              {/each}
            </select>
          </div>
        </div>
        <p class="copy-warning">⚠️ This will replace all variables in the target environment.</p>
      </div>
      <div class="modal-footer">
        <button class="btn-secondary" on:click={() => showCopyEnvironmentModal = false}>Cancel</button>
        <button class="btn-primary" on:click={handleCopyEnvironmentVariables}>Copy Variables</button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Theme System - CSS Custom Properties */
  :global(:root) {
    /* Light Theme (Default) */
    --bg-primary: #ffffff;
    --bg-secondary: #f9fafb;
    --bg-tertiary: #f3f4f6;
    --bg-accent: #ffecd0;
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
    --bg-tertiary: #585858;
    --bg-accent: #5c6c99;
    --tabs-container-bg: #374151;
    --request-item-bg: #5c5c5c;
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

  .collection-header   h2 {
    margin: 0;
    color: var(--text-primary);
    font-size: 1.125rem;
  }

  /* Theme Selector */
  .theme-selector {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .theme-selector label {
    font-size: 0.75rem;
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
    font-size: 0.75rem;
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
    font-size: 0.75rem;
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
    padding: 0.375rem 0.75rem;
    border-radius: 6px;
    font-size: 0.75rem;
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

  /* Dark theme overrides for btn-add */
  :global([data-theme="dark"]) .btn-add:hover {
    box-shadow: 0 2px 4px rgba(59, 130, 246, 0.3);
  }

  /* Dark theme overrides for variables */
  :global([data-theme="dark"]) .variable-item {
    background: var(--bg-tertiary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .variable-item:hover {
    background: var(--bg-secondary);
    border-color: var(--border-accent);
  }

  :global([data-theme="dark"]) .variable-key,
  :global([data-theme="dark"]) .variable-value {
    background: var(--bg-primary);
    color: var(--text-primary);
    border-color: var(--border-primary);
  }

  :global([data-theme="dark"]) .variable-key:focus,
  :global([data-theme="dark"]) .variable-value:focus {
    border-color: var(--border-accent);
    box-shadow: 0 0 0 2px rgba(96, 165, 250, 0.2);
  }

  :global([data-theme="dark"]) .variable-usage {
    color: var(--text-secondary);
  }

  :global([data-theme="dark"]) .variable-usage code {
    background: var(--bg-accent);
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .variable-delete-btn {
    background: var(--error);
    color: white;
  }

  :global([data-theme="dark"]) .variable-delete-btn:hover {
    background: #dc2626;
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
    font-size: 0.75rem;
    margin-top: 0.5rem;
    color: #6b7280;
  }

  .requests-list {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
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
    padding: 0.5rem;
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
    padding: 0.375rem;
    cursor: pointer;
    font-size: 0.75rem;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 28px;
    min-width: 28px;
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
    gap: 0.375rem;
    margin-bottom: 0.25rem;
  }

  .method-badge {
    padding: 0.1rem 0.3rem;
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
    color: var(--text-primary);
    font-size: 0.75rem;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .rename-input {
    flex: 1;
    font-weight: 500;
    color: var(--text-primary);
    font-size: 0.75rem;
    border: 1px solid #3b82f6;
    border-radius: 4px;
    padding: 0.25rem 0.5rem;
    background: var(--bg-primary);
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
    color: var(--text-secondary);
    margin-bottom: 0.5rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .last-response {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.375rem;
  }

  .status-badge {
    padding: 0.1rem 0.3rem;
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

  .request-description {
    font-size: 0.625rem;
    color: var(--text-secondary);
    font-style: italic;
    margin-top: 0.25rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: help;
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

  .variable-key {
    flex: 0 0 35%;
    padding: 0.5rem;
    border: 1px solid #d1d5db;
    border-radius: 4px;
    font-size: 0.75rem;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    transition: all 0.2s ease;
  }

  .variable-value {
    flex: 1;
    padding: 0.5rem;
    border: 1px solid #d1d5db;
    border-radius: 4px;
    font-size: 0.75rem;
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

  /* Environment Management Styles */
  .environment-section {
    margin-bottom: 1.5rem;
    padding: 1rem;
    background: var(--bg-secondary);
    border-radius: 8px;
    border: 1px solid var(--border-primary);
  }

  .environment-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .environment-header h3 {
    margin: 0;
    color: var(--text-primary);
    font-size: 0.875rem;
    font-weight: 600;
  }

  .environment-actions {
    display: flex;
    gap: 0.5rem;
  }

  .btn-env {
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    border: 1px solid;
    background: transparent;
    font-size: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .btn-create {
    color: var(--text-accent);
    border-color: var(--border-accent);
  }

  .btn-create:hover {
    background: var(--bg-accent);
  }

  .btn-copy {
    color: #0ea5e9;
    border-color: #0ea5e9;
  }

  .btn-copy:hover {
    background: #e0f2fe;
  }

  .btn-delete {
    color: #dc2626;
    border-color: #dc2626;
  }

  .btn-delete:hover {
    background: #fee2e2;
  }

  .environment-selector {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .environment-selector label {
    font-size: 0.75rem;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .environment-selector select {
    flex: 1;
    padding: 0.5rem;
    border: 1px solid var(--border-primary);
    border-radius: 4px;
    background: var(--bg-primary);
    color: var(--text-primary);
    font-size: 0.75rem;
  }

  .variables-section {
    margin-top: 1rem;
  }

  .variables-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .variables-header h3 {
    margin: 0;
    color: var(--text-primary);
    font-size: 0.875rem;
    font-weight: 600;
  }

  /* Modal Styles */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: var(--bg-primary);
    border-radius: 8px;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
    min-width: 400px;
    max-width: 500px;
    max-height: 80vh;
    overflow: auto;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem;
    border-bottom: 1px solid var(--border-primary);
  }

  .modal-header h3 {
    margin: 0;
    color: var(--text-primary);
    font-size: 1rem;
  }

  .modal-close {
    background: none;
    border: none;
    font-size: 1.125rem;
    cursor: pointer;
    color: var(--text-secondary);
    padding: 0;
    width: 2rem;
    height: 2rem;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
  }

  .modal-close:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }

  .modal-body {
    padding: 1.5rem;
  }

  .modal-body label {
    display: block;
    margin-bottom: 0.5rem;
    color: var(--text-primary);
    font-weight: 500;
  }

  .modal-body input,
  .modal-body select {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid var(--border-primary);
    border-radius: 4px;
    background: var(--bg-primary);
    color: var(--text-primary);
    font-size: 0.875rem;
    box-sizing: border-box;
  }

  .modal-body input:focus,
  .modal-body select:focus {
    outline: none;
    border-color: var(--border-accent);
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    padding: 1rem;
    border-top: 1px solid var(--border-primary);
  }

  .copy-form {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .form-group {
    flex: 1;
  }

  .copy-arrow {
    font-size: 1.125rem;
    color: var(--text-secondary);
    margin-top: 1.5rem;
  }

  .copy-warning {
    background: #fef3cd;
    color: #997404;
    padding: 0.75rem;
    border-radius: 4px;
    margin: 0;
    font-size: 0.75rem;
  }

  /* Group Filter Section */
  .group-filter-section {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.75rem;
    padding: 0.5rem;
    background: var(--bg-accent);
    border-radius: 6px;
    border: 1px solid var(--border-primary);
  }

  .group-selector {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .group-selector label {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .group-select {
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-primary);
    border-radius: 4px;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
    min-width: 120px;
  }

  .group-select:hover {
    border-color: var(--border-accent);
  }

  .group-select:focus {
    outline: none;
    border-color: var(--border-accent);
    box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
  }

  .group-actions {
    display: flex;
    gap: 0.5rem;
  }

  .btn-group-delete {
    background: #fee2e2;
    color: #dc2626;
    border: 1px solid #fecaca;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
    font-weight: 500;
    transition: all 0.2s ease;
  }

  .btn-group-delete:hover {
    background: #fecaca;
    border-color: #dc2626;
  }

  .tab-content-header {
    display: flex;
    justify-content: flex-start;
    align-items: center;
    margin-bottom: 1rem;
    padding: 0 0.5rem;
    gap: 0.5rem;
  }

  /* Environment Section Styles */
  .environment-section {
    background: var(--bg-tertiary, #f8fafc);
    border: 1px solid var(--border-primary, #e2e8f0);
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1rem;
  }

  .environment-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.75rem;
  }

  .environment-header h3 {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .environment-actions {
    display: flex;
    gap: 0.5rem;
  }

  .btn-env {
    background: var(--bg-primary, white);
    color: var(--text-primary);
    border: 1px solid var(--border-primary, #d1d5db);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
    font-weight: 500;
    transition: all 0.2s ease;
  }

  .btn-env:hover {
    background: var(--bg-secondary, #f9fafb);
    border-color: var(--border-accent, #667eea);
  }

  .btn-create {
    background: #ecfdf5;
    color: #059669;
    border-color: #a7f3d0;
  }

  .btn-create:hover {
    background: #d1fae5;
    border-color: #059669;
  }

  .btn-copy {
    background: #eff6ff;
    color: #2563eb;
    border-color: #bfdbfe;
  }

  .btn-copy:hover {
    background: #dbeafe;
    border-color: #2563eb;
  }

  .btn-delete {
    background: #fee2e2;
    color: #dc2626;
    border-color: #fecaca;
  }

  .btn-delete:hover {
    background: #fecaca;
    border-color: #dc2626;
  }

  .environment-selector {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .environment-selector label {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .environment-selector select {
    background: var(--bg-primary, white);
    color: var(--text-primary);
    border: 1px solid var(--border-primary, #d1d5db);
    border-radius: 4px;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
    min-width: 150px;
  }

  .environment-selector select:hover {
    border-color: var(--border-accent, #667eea);
  }

  .environment-selector select:focus {
    outline: none;
    border-color: var(--border-accent, #667eea);
    box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.1);
  }

  /* Dark theme overrides for environment section */
  :global([data-theme="dark"]) .environment-section {
    background: var(--bg-secondary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .btn-env {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .btn-env:hover {
    background: var(--bg-primary);
    border-color: var(--border-accent);
  }
</style>