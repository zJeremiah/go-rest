<script>
  import { createEventDispatcher, onMount } from 'svelte';
  
  const dispatch = createEventDispatcher();
  export let loading = false;
  export let selectedRequest = null;
  export let canSend = false;
  export let variables = []; // Add variables prop
  export let groups = []; // Groups for selection
  export let savedRequests = []; // Add savedRequests for response variable resolution
  export let hideBasicFields = false; // Hide method, URL, name when used in split layout

  let url = '';
  let method = 'GET';
  let headers = [{ key: 'Content-Type', value: 'application/json', enabled: true }];
  let body = '';
  let bodyType = 'text'; // 'text', 'json', 'form'

  // Sync internal variables with selectedRequest when changed from header
  $: if (selectedRequest && selectedRequest.url !== undefined && url !== selectedRequest.url) {
    url = selectedRequest.url;
  }
  $: if (selectedRequest && selectedRequest.method !== undefined && method !== selectedRequest.method) {
    method = selectedRequest.method;
  }
  let jsonFields = [{ key: '', value: '', enabled: true }];
  let formFields = [{ key: '', value: '', enabled: true }];
  let description = '';
  let activeTab = 'headers';
  let params = [{ key: '', value: '', enabled: true }];
  let saveStatus = ''; // Track save status for user feedback


  const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];
  const bodyTypes = [
    { id: 'text', label: 'Text' },
    { id: 'json', label: 'JSON' },
    { id: 'form', label: 'Form URL Encoded' }
  ];
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
  let showPreviewModal = false;
  let previewModalType = ''; // 'headers', 'body'
  let previewModalTitle = '';
  let previewModalContent = {};
  
  function updateHeadersPreview() {
    previewHeaders = buildHeadersPreview();
  }

  // Update headers preview when headers, variables, or savedRequests change
  $: if (headers || variables || savedRequests) updateHeadersPreview();



  // Build body with variable substitution for preview
  function buildBodyPreview() {
    if (bodyType === 'text') {
      return processTemplate(body, variables);
    } else if (bodyType === 'json') {
      const processedFields = {};
      jsonFields.filter(f => f.enabled && f.key && f.key.trim()).forEach(field => {
        const processedKey = processTemplate(field.key.trim(), variables);
        const processedValue = processTemplate(field.value || '', variables);
        processedFields[processedKey] = processedValue;
      });
      return JSON.stringify(processedFields, null, 2);
    } else if (bodyType === 'form') {
      const processedForm = {};
      formFields.filter(f => f.enabled && f.key && f.key.trim()).forEach(field => {
        const processedKey = processTemplate(field.key.trim(), variables);
        const processedValue = processTemplate(field.value || '', variables);
        processedForm[processedKey] = processedValue;
      });
      return processedForm;
    }
    return '';
  }

  // Check if content has variables
  function hasVariables(content) {
    if (typeof content === 'string') {
      return content.includes('{{') && content.includes('}}');
    } else if (typeof content === 'object') {
      return Object.values(content).some(value => 
        typeof value === 'string' && value.includes('{{') && value.includes('}}')
      );
    }
    return false;
  }

  // Open preview modal
  function openPreviewModal(type) {
    previewModalType = type;
    
    if (type === 'headers') {
      previewModalTitle = 'Preview Headers';
      previewModalContent = buildHeadersPreview();
    } else if (type === 'body') {
      previewModalTitle = 'Preview Request Body';
      const bodyPreview = buildBodyPreview();
      if (typeof bodyPreview === 'string') {
        previewModalContent = { '_raw_': bodyPreview };
      } else {
        previewModalContent = bodyPreview;
      }
    }
    
    showPreviewModal = true;
  }

  // Close preview modal
  function closePreviewModal() {
    showPreviewModal = false;
  }

  // Process template variables in a string (frontend version with response variable support)
  function processTemplate(input, vars) {
    if (!input) return input;
    
    // Convert input to string if it's an object
    let result = typeof input === 'string' ? input : 
                 typeof input === 'object' && input !== null ? JSON.stringify(input) : 
                 String(input);
    
    // First, handle response variables
    const responseVarPattern = /\{\{[^}]*\}\}/g;
    const allMatches = result.match(responseVarPattern) || [];
    
    for (const match of allMatches) {
      // Check if this looks like a response variable (contains quotes)
      if (match.includes('"') || match.includes('\\"')) {
        const resolvedValue = resolveResponseVariable(match);
        if (resolvedValue !== null) {
          result = result.replace(match, resolvedValue);
        }
      }
    }
    
    // Then handle regular environment variables
    if (vars) {
      vars.forEach(variable => {
        if (variable.key && variable.value !== undefined) {
          const regex = new RegExp(`{{\\s*${variable.key}\\s*}}`, 'g');
          // Use resolved value for environment variables, fallback to raw value
          const valueToUse = variable.isEnvVar && variable.resolvedValue ? variable.resolvedValue : variable.value;
          result = result.replace(regex, valueToUse);
        }
      });
    }
    
    return result;
  }

  // Resolve response variables like {{"RequestName".field}} or {{\"RequestName\".field}}
  function resolveResponseVariable(variable) {
    try {
      // Remove outer {{ and }}
      const content = variable.slice(2, -2).trim();
      
      // Handle both escaped and unescaped quotes
      let requestName, fieldPath;
      
      if (content.startsWith('\\"')) {
        // Handle escaped quotes: {{\"RequestName\".field}}
        const endIndex = content.indexOf('\\".', 2);
        if (endIndex === -1) {
          return null;
        }
        requestName = content.substring(2, endIndex);
        fieldPath = content.substring(endIndex + 3);
      } else if (content.startsWith('"')) {
        // Handle regular quotes: {{"RequestName".field}}
        const endIndex = content.indexOf('".', 1);
        if (endIndex === -1) {
          return null;
        }
        requestName = content.substring(1, endIndex);
        fieldPath = content.substring(endIndex + 2);
      } else {
        return null; // Not a response variable
      }
      
      // Find the request
      const request = savedRequests.find(r => r.name === requestName);
      if (!request) {
        return ''; // Return empty string if request not found
      }
      
      if (!request.lastResponse) {
        return ''; // Return empty string if no response
      }
      
      // Extract the field value
      return extractFieldFromResponse(request.lastResponse.body, fieldPath);
    } catch (error) {
      return null;
    }
  }

  // Extract field from response body (client-side version)
  function extractFieldFromResponse(responseBody, fieldPath) {
    if (!responseBody) return '';
    
    // If requesting full response
    if (fieldPath === 'response') {
      if (typeof responseBody === 'string') {
        return responseBody;
      }
      return JSON.stringify(responseBody);
    }
    
    // Navigate JSON structure
    let current = responseBody;
    const parts = fieldPath.split('.');
    
    for (const part of parts) {
      if (!part) continue;
      
      if (current && typeof current === 'object' && part in current) {
        current = current[part];
      } else {
        return ''; // Field doesn't exist
      }
    }
    
    // Convert to string
    if (typeof current === 'string') {
      return current;
    } else if (current === null || current === undefined) {
      return '';
    } else {
      return JSON.stringify(current);
    }
  }

  // Generate tooltip text for a variable name
  function getVariableTooltip(variableName, vars) {
    if (!variableName || !vars) return '';
    
    const variable = vars.find(v => v.key === variableName.trim());
    if (variable && variable.value !== undefined && variable.value !== '') {
      return `${variableName}: ${variable.value}`;
    } else {
      return `${variableName}: undefined`;
    }
  }

  // Check if a string contains variables and return tooltip info
  function analyzeVariables(input, vars, requests = []) {
    if (!input) return { hasVariables: false, tooltip: '' };
    
    // Convert input to string if it's an object
    const inputString = typeof input === 'string' ? input : 
                       typeof input === 'object' && input !== null ? JSON.stringify(input) : 
                       String(input);
    
    const variableMatches = inputString.match(/\{\{\s*([^}]+)\s*\}\}/g);
    if (!variableMatches) return { hasVariables: false, tooltip: '' };
    
    const tooltips = variableMatches.map(match => {
      const content = match.replace(/[{}]/g, '').trim();
      
      // Check if it's a response variable (starts with quote)
      if (content.startsWith('"')) {
        return getResponseVariableTooltip(content, requests);
      } else {
        // Regular environment variable
        return getVariableTooltip(content, vars || []);
      }
    });
    
    return {
      hasVariables: true,
      tooltip: tooltips.join(' | '),
      variableCount: variableMatches.length
    };
  }

  // Generate tooltip text for response variables
  function getResponseVariableTooltip(content, requests = []) {
    try {
      // Parse: "RequestName".field
      const quoteMatch = content.match(/^"([^"\\]*(?:\\.[^"\\]*)*)"\.(.+)$/);
      if (!quoteMatch) {
        return content + ': invalid response variable format';
      }
      
      const requestName = quoteMatch[1].replace(/\\"/g, '"'); // Unescape quotes
      const fieldPath = quoteMatch[2];
      
      // Try to find the request and get the field value for preview
      const request = requests.find(r => r.name === requestName);
      if (!request) {
        return `${requestName}.${fieldPath}: request not found`;
      }
      
      if (!request.lastResponse) {
        return `${requestName}.${fieldPath}: no response data`;
      }
      
      // Extract the field value for preview
      const value = extractFieldFromResponse(request.lastResponse.body, fieldPath);
      const preview = value && value.length > 50 ? value.substring(0, 47) + '...' : value;
      
      return `${requestName}.${fieldPath}: ${preview || 'undefined'}`;
    } catch (error) {
      return content + ': error parsing response variable';
    }
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

  // JSON field management
  function addJsonField() {
    jsonFields = [...jsonFields, { key: '', value: '', enabled: true }];
  }

  function removeJsonField(index) {
    jsonFields = jsonFields.filter((_, i) => i !== index);
  }

  // Form field management
  function addFormField() {
    formFields = [...formFields, { key: '', value: '', enabled: true }];
  }

  function removeFormField(index) {
    formFields = formFields.filter((_, i) => i !== index);
  }

  // Build body content based on type
  function buildBodyContent() {
    switch (bodyType) {
      case 'json':
        const jsonObj = {};
        jsonFields.filter(f => f.enabled && f.key && f.key.trim()).forEach(field => {
          const processedKey = processTemplate(field.key.trim(), variables);
          let processedValue = processTemplate(field.value || '', variables);
          
          // Try to parse the processed value as JSON if it looks like JSON
          try {
            if (processedValue && (processedValue.startsWith('{') || processedValue.startsWith('[') || processedValue === 'true' || processedValue === 'false' || processedValue === 'null' || !isNaN(processedValue))) {
              processedValue = JSON.parse(processedValue);
            }
          } catch (e) {
            // Keep as string if not valid JSON
          }
          
          jsonObj[processedKey] = processedValue;
        });
        return JSON.stringify(jsonObj, null, 2);
      
      case 'form':
        const formData = new URLSearchParams();
        formFields.filter(f => f.enabled && f.key && f.key.trim()).forEach(field => {
          const processedKey = processTemplate(field.key.trim(), variables);
          const processedValue = processTemplate(field.value || '', variables);
          formData.append(processedKey, processedValue);
        });
        return formData.toString();
      
      case 'text':
      default:
        return processTemplate(body, variables);
    }
  }

  // Build raw body content (without template processing) for saving
  function buildRawBodyContent() {
    switch (bodyType) {
      case 'json':
        const jsonObj = {};
        jsonFields.filter(f => f.enabled && f.key && f.key.trim()).forEach(field => {
          const key = field.key.trim();
          let value = field.value || '';
          
          // Try to parse the value as JSON if it looks like JSON (but don't process templates)
          try {
            if (value && (value.startsWith('{') || value.startsWith('[') || value === 'true' || value === 'false' || value === 'null' || !isNaN(value))) {
              value = JSON.parse(value);
            }
          } catch (e) {
            // Keep as string if not valid JSON
          }
          
          jsonObj[key] = value;
        });
        return JSON.stringify(jsonObj, null, 2);
      
      case 'form':
        const formData = new URLSearchParams();
        formFields.filter(f => f.enabled && f.key && f.key.trim()).forEach(field => {
          formData.append(field.key.trim(), field.value || '');
        });
        return formData.toString();
      
      case 'text':
      default:
        return body;
    }
  }

  // Parse body content when switching types
  function parseBodyContent(content, targetType) {
    try {
      switch (targetType) {
        case 'json':
          if (content && content.trim()) {
            const parsed = JSON.parse(content);
            if (typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed)) {
              return Object.entries(parsed).map(([key, value]) => ({
                key,
                value: typeof value === 'string' ? value : JSON.stringify(value),
                enabled: true
              }));
            }
          }
          return [{ key: '', value: '', enabled: true }];
        
        case 'form':
          if (content && content.trim()) {
            const params = new URLSearchParams(content);
            const fields = [];
            for (const [key, value] of params.entries()) {
              fields.push({ key, value, enabled: true });
            }
            return fields.length > 0 ? fields : [{ key: '', value: '', enabled: true }];
          }
          return [{ key: '', value: '', enabled: true }];
        
        case 'text':
        default:
          return content;
      }
    } catch (e) {
      // If parsing fails, return default
      if (targetType === 'json' || targetType === 'form') {
        return [{ key: '', value: '', enabled: true }];
      }
      return content;
    }
  }

  // Handle body type change - preserve template variables by using raw content
  function handleBodyTypeChange() {
    // Don't convert content between types - just preserve each type independently
    // Each body type maintains its own content and we simply switch between them
    
    // Update Content-Type header based on body type
    const contentTypeIndex = headers.findIndex(h => h.key.toLowerCase() === 'content-type');
    if (contentTypeIndex !== -1) {
      // Update existing Content-Type header
      if (bodyType === 'json') {
        headers[contentTypeIndex].value = 'application/json';
      } else if (bodyType === 'form') {
        headers[contentTypeIndex].value = 'application/x-www-form-urlencoded';
      }
      headers = [...headers]; // Trigger reactivity
    } else {
      // Add Content-Type header if it doesn't exist
      if (bodyType === 'json') {
        headers = [...headers, { key: 'Content-Type', value: 'application/json', enabled: true }];
      } else if (bodyType === 'form') {
        headers = [...headers, { key: 'Content-Type', value: 'application/x-www-form-urlencoded', enabled: true }];
      }
    }
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

  // Initialize preview on component mount and when variables/URL/savedRequests change
  $: if (url || variables || savedRequests) updatePreview();

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
    
    const currentJsonFields = jsonFields.filter(f => f.key && f.key.trim());
    const lastJsonFields = lastSavedState.jsonFields || [];
    
    const currentFormFields = formFields.filter(f => f.key && f.key.trim());
    const lastFormFields = lastSavedState.formFields || [];
    
    const currentBodyContent = buildRawBodyContent();
    const lastBodyContent = lastSavedState.bodyContent || '';
    
    // Normalize content for comparison (handle objects, strings, and formatting)
    const normalizeContent = (content) => {
      if (content === null || content === undefined) return '';
      if (typeof content === 'object') {
        return JSON.stringify(content);
      }
      if (typeof content === 'string' && content.trim().startsWith('{')) {
        try {
          return JSON.stringify(JSON.parse(content));
        } catch {
          return content;
        }
      }
      return String(content);
    };
    
    return (
      url !== lastSavedState.url ||
      method !== lastSavedState.method ||
      JSON.stringify(currentHeaders) !== JSON.stringify(lastHeaders) ||
      normalizeContent(currentBodyContent) !== normalizeContent(lastBodyContent) ||
      bodyType !== lastSavedState.bodyType ||
      body !== (lastSavedState.bodyText || '') ||
      JSON.stringify(currentJsonFields) !== JSON.stringify(lastJsonFields) ||
      JSON.stringify(currentFormFields) !== JSON.stringify(lastFormFields) ||
      JSON.stringify(currentParams) !== JSON.stringify(lastParams) ||
      (description || '') !== (lastSavedState.description || '')
    );
  }

  // Auto-save when any form field changes (explicitly depend on all form fields)
  $: if (selectedRequest && url && url.trim() && !isLoadingRequest && (url || method || headers || body || bodyType || jsonFields || formFields || params || description)) {
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

  // Save immediately when parameter fields lose focus - no conditions, just save
  function handleParamFieldBlur() {
    if (selectedRequest) {
      clearTimeout(urlSaveTimeout);
      saveStatus = 'saving';
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
    const finalBodyContent = buildBodyContent();
    const requestData = {
      url: processedUrl || url.trim(),
      method,
      headers: parsedHeaders,
      body: finalBodyContent,
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
    // Save ALL body types independently - preserve each type's current content
    const requestData = {
      url: url.trim(), // Save raw URL with template variables intact
      method,
      headers: parsedHeaders,
      body: buildRawBodyContent(), // Update legacy body field with active body type for compatibility
      bodyType: bodyType, // Save which body type is active
      bodyText: body, // Always save current text body content
      bodyJson: jsonFields.filter(f => f.key && f.key.trim()), // Always save current JSON fields
      bodyForm: formFields.filter(f => f.key && f.key.trim()), // Always save current form fields
      params: params.filter(p => p.key && p.key.trim()),
      group: selectedRequest?.group || 'default',
      description: description.trim()
    };


    
    // Update our saved state tracking
    lastSavedState = {
      url: url,
      method: method,
      headers: [...headers.filter(h => h.key && h.key.trim())],
      bodyContent: buildRawBodyContent(),
      bodyType: bodyType,
      bodyText: body,
      jsonFields: [...jsonFields.filter(f => f.key && f.key.trim())],
      formFields: [...formFields.filter(f => f.key && f.key.trim())],
      params: [...params.filter(p => p.key && p.key.trim())],
      description: description
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
    description = data.description || '';
    
    // Handle body content with separate storage for each type
    if (data.bodyType && (data.bodyText !== undefined || data.bodyJson || data.bodyForm)) {
      // New format: separate storage for each body type
      bodyType = data.bodyType || 'text';
      body = data.bodyText || '';
      jsonFields = data.bodyJson && data.bodyJson.length > 0 ? data.bodyJson : [{ key: '', value: '', enabled: true }];
      formFields = data.bodyForm && data.bodyForm.length > 0 ? data.bodyForm : [{ key: '', value: '', enabled: true }];
    } else {
      // Legacy format: detect body type and parse from single body field
      let bodyContent = '';
      if (data.body) {
        if (typeof data.body === 'object' && data.body !== null) {
          bodyContent = JSON.stringify(data.body, null, 2);
          body = bodyContent;
        } else {
          bodyContent = String(data.body).trim();
          body = data.body;
        }
      } else {
        body = '';
      }
      
      // Detect body type and parse fields
      if (bodyContent) {
        // Try to detect if it's JSON
        try {
          const parsed = JSON.parse(bodyContent);
          if (typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed)) {
            bodyType = 'json';
            jsonFields = Object.entries(parsed).map(([key, value]) => ({
              key,
              value: typeof value === 'string' ? value : JSON.stringify(value),
              enabled: true
            }));
            formFields = [{ key: '', value: '', enabled: true }]; // Reset form fields
          } else {
            bodyType = 'text';
            jsonFields = [{ key: '', value: '', enabled: true }]; // Reset JSON fields
            formFields = [{ key: '', value: '', enabled: true }]; // Reset form fields
          }
        } catch (e) {
          // Try to detect if it's form URL encoded
          try {
            const params = new URLSearchParams(bodyContent);
            const hasParams = Array.from(params.entries()).length > 0;
            if (hasParams) {
              bodyType = 'form';
              formFields = Array.from(params.entries()).map(([key, value]) => ({
                key,
                value,
                enabled: true
              }));
              jsonFields = [{ key: '', value: '', enabled: true }]; // Reset JSON fields
            } else {
              bodyType = 'text';
              jsonFields = [{ key: '', value: '', enabled: true }]; // Reset JSON fields
              formFields = [{ key: '', value: '', enabled: true }]; // Reset form fields
            }
          } catch (e) {
            bodyType = 'text';
            jsonFields = [{ key: '', value: '', enabled: true }]; // Reset JSON fields
            formFields = [{ key: '', value: '', enabled: true }]; // Reset form fields
          }
        }
      } else {
        bodyType = 'text';
        jsonFields = [{ key: '', value: '', enabled: true }];
        formFields = [{ key: '', value: '', enabled: true }];
      }
    }
    
    // Handle headers - convert from object to array format
    if (data.headers && Object.keys(data.headers).length > 0) {
      headers = Object.entries(data.headers).map(([key, value]) => ({
        key: key,
        value: value,
        enabled: true
      }));
    } else {
      // Set default Content-Type based on detected body type
      let defaultContentType = 'application/json';
      if (bodyType === 'form') {
        defaultContentType = 'application/x-www-form-urlencoded';
      }
      headers = [{ key: 'Content-Type', value: defaultContentType, enabled: true }];
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
    const initialBodyContent = buildRawBodyContent();
    lastSavedState = {
      url: newUrl,
      method: method,
      headers: [...headers.filter(h => h.key && h.key.trim())],
      bodyContent: initialBodyContent,
      bodyType: bodyType,
      bodyText: body,
      jsonFields: [...jsonFields.filter(f => f.key && f.key.trim())],
      formFields: [...formFields.filter(f => f.key && f.key.trim())],
      params: [...params.filter(p => p.key && p.key.trim())],
      description: description
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

  // Export functions so parent can call them
  export { saveCurrentRequest, handleSubmit };
</script>

<div class="card">
  {#if !hideBasicFields}
    <h2 class="request-section-title">🔄 HTTP Request</h2>
    
    {#if selectedRequest && selectedRequest.name}
      <div class="selected-request-header">
        <div class="request-name-display">
          <h3 class="request-name">{selectedRequest.name}</h3>
          {#if selectedRequest.group && selectedRequest.group !== 'default'}
            <span class="request-group">({selectedRequest.group})</span>
          {/if}
        </div>
      </div>
    {/if}
  {:else}
    <h2 class="request-section-title">⚙️ Request Options</h2>
  {/if}
  
  <form on:submit|preventDefault={handleSubmit}>
    {#if !hideBasicFields}
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
                                class="input {analyzeVariables(url, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
                      title={analyzeVariables(url, variables, savedRequests).hasVariables ? analyzeVariables(url, variables, savedRequests).tooltip : ''}
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
    {/if}

    {#if !hideBasicFields}
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
    {:else}
      <!-- Save button for options-only mode -->
      <div class="button-row">
        <button 
          type="button" 
          class="btn btn-secondary btn-full-width" 
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
      </div>
    {/if}
    
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
                                    class="input header-input {analyzeVariables(header.key, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
              title={analyzeVariables(header.key, variables, savedRequests).hasVariables ? analyzeVariables(header.key, variables, savedRequests).tooltip : ''}
                    />
                    <input 
                      type="text" 
                      bind:value={header.value}
                      on:blur={handleFieldBlur}
                      placeholder="Header Value"
                                    class="input header-input {analyzeVariables(header.value, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
              title={analyzeVariables(header.value, variables, savedRequests).hasVariables ? analyzeVariables(header.value, variables, savedRequests).tooltip : ''}
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

              <!-- Headers Preview Button -->
              {#if Object.keys(previewHeaders).length > 0 && hasHeaderVariables()}
                <div class="preview-button-container">
                  <button 
                    type="button" 
                    class="btn-preview" 
                    on:click={() => openPreviewModal('headers')}
                    title="Preview headers with variables resolved"
                  >
                    👁️ Preview Headers
                  </button>
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
                      on:change={(e) => {
                        updatePreview();
                        handleParamFieldBlur();
                      }}
                    />
                    <input 
                      type="text" 
                      bind:value={param.key}
                      on:blur={handleParamFieldBlur}
                      placeholder="Key"
                                    class="input param-input {analyzeVariables(param.key, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
              title={analyzeVariables(param.key, variables, savedRequests).hasVariables ? analyzeVariables(param.key, variables, savedRequests).tooltip : ''}
                      on:input={updatePreview}
                    />
                    <input 
                      type="text" 
                      bind:value={param.value}
                      on:blur={handleParamFieldBlur}
                      placeholder="Value"
                                    class="input param-input {analyzeVariables(param.value, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
              title={analyzeVariables(param.value, variables, savedRequests).hasVariables ? analyzeVariables(param.value, variables, savedRequests).tooltip : ''}
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

    <div class="form-group">
      <label for="body">Request Body</label>
        
        <!-- Body Type Selector -->
        <div class="body-type-selector">
          {#each bodyTypes as type}
            <label class="body-type-option">
              <input 
                type="radio" 
                bind:group={bodyType} 
                value={type.id}
                on:change={handleBodyTypeChange}
              />
              <span>{type.label}</span>
            </label>
          {/each}
        </div>

        <!-- Text Body -->
        {#if bodyType === 'text'}
          <textarea 
            id="body"
            bind:value={body} 
            on:blur={handleFieldBlur}
            placeholder="Raw text, JSON, XML, etc..."
            class="textarea {analyzeVariables(body, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
            title={analyzeVariables(body, variables, savedRequests).hasVariables ? analyzeVariables(body, variables, savedRequests).tooltip : ''}
            rows="6"
          ></textarea>
          
          <!-- Text Body Preview Button -->
          {#if body && hasVariables(body)}
            <div class="preview-button-container">
              <button 
                type="button" 
                class="btn-preview" 
                on:click={() => openPreviewModal('body')}
                title="Preview request body with variables resolved"
              >
                👁️ Preview Body
              </button>
            </div>
          {/if}
        {/if}

        <!-- JSON Body -->
        {#if bodyType === 'json'}
          <div class="json-fields">
            {#each jsonFields as field, index}
              <div class="field-row">
                <input 
                  type="checkbox" 
                  bind:checked={field.enabled}
                  class="field-checkbox"
                />
                <input 
                  type="text" 
                  bind:value={field.key}
                  placeholder="Key"
                  class="field-input field-key {analyzeVariables(field.key, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
                  title={analyzeVariables(field.key, variables, savedRequests).hasVariables ? analyzeVariables(field.key, variables, savedRequests).tooltip : ''}
                />
                <input 
                  type="text" 
                  bind:value={field.value}
                  placeholder="Value"
                  class="field-input field-value {analyzeVariables(field.value, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
                  title={analyzeVariables(field.value, variables, savedRequests).hasVariables ? analyzeVariables(field.value, variables, savedRequests).tooltip : ''}
                />
                <button 
                  type="button" 
                  class="btn-remove-field" 
                  on:click={() => removeJsonField(index)}
                  disabled={jsonFields.length <= 1}
                >
                  ✕
                </button>
              </div>
            {/each}
            <button type="button" class="btn-add-field" on:click={addJsonField}>
              + Add JSON Field
            </button>
          </div>
          
          <!-- JSON Body Preview Button -->
          {#if jsonFields.some(f => f.enabled && (hasVariables(f.key) || hasVariables(f.value)))}
            <div class="preview-button-container">
              <button 
                type="button" 
                class="btn-preview" 
                on:click={() => openPreviewModal('body')}
                title="Preview JSON body with variables resolved"
              >
                👁️ Preview Body
              </button>
            </div>
          {/if}
        {/if}

        <!-- Form URL Encoded Body -->
        {#if bodyType === 'form'}
          <div class="form-fields">
            {#each formFields as field, index}
              <div class="field-row">
                <input 
                  type="checkbox" 
                  bind:checked={field.enabled}
                  class="field-checkbox"
                />
                <input 
                  type="text" 
                  bind:value={field.key}
                  placeholder="Key"
                  class="field-input field-key {analyzeVariables(field.key, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
                  title={analyzeVariables(field.key, variables, savedRequests).hasVariables ? analyzeVariables(field.key, variables, savedRequests).tooltip : ''}
                />
                <input 
                  type="text" 
                  bind:value={field.value}
                  placeholder="Value"
                  class="field-input field-value {analyzeVariables(field.value, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
                  title={analyzeVariables(field.value, variables, savedRequests).hasVariables ? analyzeVariables(field.value, variables, savedRequests).tooltip : ''}
                />
                <button 
                  type="button" 
                  class="btn-remove-field" 
                  on:click={() => removeFormField(index)}
                  disabled={formFields.length <= 1}
                >
                  ✕
                </button>
              </div>
            {/each}
            <button type="button" class="btn-add-field" on:click={addFormField}>
              + Add Form Field
            </button>
          </div>
          
          <!-- Form Body Preview Button -->
          {#if formFields.some(f => f.enabled && (hasVariables(f.key) || hasVariables(f.value)))}
            <div class="preview-button-container">
              <button 
                type="button" 
                class="btn-preview" 
                on:click={() => openPreviewModal('body')}
                title="Preview form body with variables resolved"
              >
                👁️ Preview Body
              </button>
            </div>
          {/if}
        {/if}
      </div>

    <div class="form-group">
      <label for="description">Description</label>
      <textarea 
        id="description"
        bind:value={description} 
        on:blur={handleFieldBlur}
        placeholder="Add a description for this request..."
        class="textarea description-textarea {analyzeVariables(description, variables, savedRequests).hasVariables ? 'has-variables' : ''}"
        title={analyzeVariables(description, variables, savedRequests).hasVariables ? analyzeVariables(description, variables, savedRequests).tooltip : ''}
        rows="3"
      ></textarea>
    </div>

  </form>
</div>

<!-- Preview Modal -->
{#if showPreviewModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div class="modal-overlay" role="dialog" aria-modal="true" on:click={closePreviewModal} on:keydown={(e) => e.key === 'Escape' && closePreviewModal()}>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal-content" role="document" on:click={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h3>{previewModalTitle}</h3>
        <button type="button" class="modal-close" on:click={closePreviewModal}>✕</button>
      </div>
      
      <div class="modal-body">
        {#if previewModalType === 'body' && previewModalContent._raw_}
          <!-- Raw text body -->
          <pre class="preview-text">{previewModalContent._raw_}</pre>
        {:else if Object.keys(previewModalContent).length > 0}
          <!-- Key-value pairs for headers, params, or structured body -->
          <div class="preview-content">
            {#each Object.entries(previewModalContent) as [key, value]}
              <div class="preview-row">
                <span class="preview-key">{key}:</span>
                <span class="preview-value">{value}</span>
              </div>
            {/each}
          </div>
        {:else}
          <p class="preview-empty">No content to preview</p>
        {/if}
      </div>
      
      <div class="modal-footer">
        <button type="button" class="btn-secondary" on:click={closePreviewModal}>Close</button>
        <button 
          type="button" 
          class="btn-primary" 
          on:click={() => {
            if (previewModalType === 'body' && previewModalContent._raw_) {
              navigator.clipboard.writeText(previewModalContent._raw_);
            } else {
              const text = Object.entries(previewModalContent)
                .map(([key, value]) => `${key}: ${value}`)
                .join('\n');
              navigator.clipboard.writeText(text);
            }
          }}
          title="Copy to clipboard"
        >
          📋 Copy
        </button>
      </div>
    </div>
  </div>
{/if}

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

  .btn-full-width {
    width: 100%;
  }



  textarea {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.8rem;
  }

  .description-textarea {
    font-family: inherit !important;
    font-size: 0.75rem !important;
    line-height: 1.4;
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
    padding: 1rem;
  }

  /* Headers Styles */
  .headers-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.75rem;
  }

  .headers-header span {
    font-weight: 500;
    color: #374151;
    margin: 0;
  }

  .headers-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-height: 300px;
    overflow-y: auto;
  }

  .header-row {
    display: grid;
    grid-template-columns: auto 1fr 1fr auto;
    gap: 0.5rem;
    align-items: center;
    padding: 0.5rem;
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



  /* Params Styles */
  .params-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.75rem;
  }

  .params-header span {
    font-weight: 500;
    color: #374151;
    margin: 0;
  }

  .params-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-height: 300px;
    overflow-y: auto;
  }

  .param-row {
    display: grid;
    grid-template-columns: auto 1fr 1fr auto;
    gap: 0.5rem;
    align-items: center;
    padding: 0.5rem;
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

  /* Enable text selection for URL preview */
  .preview-text-top,
  .url-preview-top,
  .url-preview-top * {
    user-select: text !important;
    -webkit-user-select: text !important;
    -moz-user-select: text !important;
    cursor: text !important;
  }

  /* Body Type Selector Styles */
  .body-type-selector {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
    padding: 0.5rem;
    background: #f9fafb;
    border-radius: 6px;
    border: 1px solid #e5e7eb;
  }

  .body-type-option {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
    font-size: 0.875rem;
    font-weight: 500;
    color: #6b7280;
  }

  .body-type-option input[type="radio"] {
    margin: 0;
  }

  .body-type-option:has(input:checked) span {
    color: #2563eb;
    font-weight: 600;
  }

  /* JSON and Form Fields Styles */
  .json-fields,
  .form-fields {
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    background: white;
  }

  .field-row {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.5rem;
    border-bottom: 1px solid #f3f4f6;
  }

  .field-row:last-of-type {
    border-bottom: none;
  }

  .field-checkbox {
    margin: 0;
    cursor: pointer;
  }

  .field-input {
    flex: 1;
    padding: 0.5rem;
    border: 1px solid #d1d5db;
    border-radius: 4px;
    font-size: 0.875rem;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  }

  .field-key {
    flex: 0 0 200px;
  }

  .field-value {
    flex: 2;
  }

  .field-input:focus {
    outline: none;
    border-color: #2563eb;
    box-shadow: 0 0 0 1px #2563eb;
  }

  .btn-remove-field {
    background: #fee2e2;
    color: #dc2626;
    border: 1px solid #fecaca;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.875rem;
    min-width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .btn-remove-field:hover:not(:disabled) {
    background: #fecaca;
    border-color: #dc2626;
  }

  .btn-remove-field:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-add-field {
    background: #f0f9ff;
    color: #0369a1;
    border: 1px solid #bae6fd;
    padding: 0.75rem;
    width: 100%;
    cursor: pointer;
    font-size: 0.875rem;
    font-weight: 500;
    transition: all 0.2s ease;
  }

  .btn-add-field:hover {
    background: #e0f2fe;
    border-color: #0369a1;
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

  :global([data-theme="dark"]) .headers-header,
  :global([data-theme="dark"]) .params-header {
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .headers-header span,
  :global([data-theme="dark"]) .params-header span {
    color: var(--text-primary) !important;
  }

  :global([data-theme="dark"]) .header-row,
  :global([data-theme="dark"]) .param-row,
  :global([data-theme="dark"]) .field-row {
    background: var(--bg-tertiary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .header-input,
  :global([data-theme="dark"]) .param-input,
  :global([data-theme="dark"]) .field-input {
    background: var(--bg-primary);
    color: var(--text-primary);
    border-color: var(--border-primary);
  }

  :global([data-theme="dark"]) .header-input:focus,
  :global([data-theme="dark"]) .param-input:focus,
  :global([data-theme="dark"]) .field-input:focus {
    border-color: var(--border-accent);
    box-shadow: 0 0 0 2px rgba(96, 165, 250, 0.2);
  }

  :global([data-theme="dark"]) .textarea {
    background: var(--bg-primary);
    color: var(--text-primary);
    border-color: var(--border-primary);
  }

  :global([data-theme="dark"]) .textarea:focus {
    border-color: var(--border-accent);
    box-shadow: 0 0 0 2px rgba(96, 165, 250, 0.2);
  }



  :global([data-theme="dark"]) .description-textarea {
    background: var(--bg-primary);
    color: var(--text-primary);
    border-color: var(--border-primary);
  }

  :global([data-theme="dark"]) .description-textarea:focus {
    border-color: var(--border-accent);
    box-shadow: 0 0 0 2px rgba(96, 165, 250, 0.2);
  }

  :global([data-theme="dark"]) .btn-remove,
  :global([data-theme="dark"]) .btn-remove-field {
    background: var(--error);
    color: white;
  }

  :global([data-theme="dark"]) .btn-remove:hover,
  :global([data-theme="dark"]) .btn-remove-field:hover {
    background: #dc2626;
  }

  :global([data-theme="dark"]) .btn-add-field {
    background: var(--button-primary);
    color: white;
    border-color: var(--button-primary);
  }

  :global([data-theme="dark"]) .btn-add-field:hover {
    background: var(--button-primary-hover);
    border-color: var(--button-primary-hover);
  }

  :global([data-theme="dark"]) .body-type-selector {
    background: var(--bg-secondary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .body-type-option {
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .body-type-option input[type="radio"]:checked + label {
    color: var(--text-accent);
  }

  /* Variable tooltip styling */
  .has-variables {
    position: relative;
  }

  .has-variables::after {
    content: '';
    position: absolute;
    right: 8px;
    top: 50%;
    transform: translateY(-50%);
    width: 6px;
    height: 6px;
    background: #3b82f6;
    border-radius: 50%;
    opacity: 0.7;
  }

  .has-variables:focus::after {
    opacity: 1;
    box-shadow: 0 0 4px #3b82f6;
  }

  /* Dark theme variable indicator */
  :global([data-theme="dark"]) .has-variables::after {
    background: #60a5fa;
  }

  :global([data-theme="dark"]) .has-variables:focus::after {
    box-shadow: 0 0 4px #60a5fa;
  }

  /* Compact header styling */
  .request-section-title {
    margin: 0 0 0.75rem 0;
    font-size: 1.25rem;
  }

  /* Compact first form group */
  .card form > .form-group:first-child {
    margin-top: 0;
  }

  /* Ensure consistent spacing when no request is selected */
  .card form {
    margin-top: 0.75rem;
  }

  /* When request is selected, reduce the form top margin */
  .selected-request-header + form {
    margin-top: 0;
  }

  /* Selected request header */
  .selected-request-header {
    margin: -0.5rem -1rem 0.5rem -1rem;
    padding: 0.5rem 1rem;
    background: var(--bg-tertiary);
    border-bottom: 1px solid var(--border-color);
    border-radius: 8px 8px 0 0;
  }

  .request-name-display {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.95rem;
  }

  .request-name {
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    font-size: 1.1rem;
  }

  .request-group {
    font-size: 0.85rem;
    color: var(--text-secondary);
    opacity: 0.8;
  }

  /* Dark theme overrides */
  :global([data-theme="dark"]) .selected-request-header {
    background: var(--bg-quaternary);
    border-bottom-color: var(--border-color);
  }

  :global([data-theme="dark"]) .request-name {
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .request-group {
    color: var(--text-secondary);
  }

  /* Preview Button Styles */
  .preview-button-container {
    margin-top: 1rem;
    display: flex;
    justify-content: flex-end;
  }

  .btn-preview {
    background: var(--button-secondary, #f3f4f6);
    color: var(--text-primary, #374151);
    border: 1px solid var(--border-primary, #d1d5db);
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    transition: all 0.2s ease;
  }

  .btn-preview:hover {
    background: var(--button-secondary-hover, #e5e7eb);
    border-color: var(--border-accent, #667eea);
  }

  :global([data-theme="dark"]) .btn-preview {
    background: var(--button-secondary);
    color: var(--text-primary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .btn-preview:hover {
    background: var(--button-secondary-hover);
    border-color: var(--border-accent);
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
    backdrop-filter: blur(2px);
  }

  .modal-content {
    background: var(--bg-primary, white);
    border-radius: 8px;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
    max-width: 90vw;
    max-height: 90vh;
    width: 600px;
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.5rem;
    border-bottom: 1px solid var(--border-secondary, #e5e7eb);
  }

  .modal-header h3 {
    margin: 0;
    color: var(--text-primary, #374151);
    font-size: 1.25rem;
    font-weight: 600;
  }

  .modal-close {
    background: none;
    border: none;
    font-size: 1.25rem;
    cursor: pointer;
    color: var(--text-secondary, #6b7280);
    padding: 0.25rem;
    border-radius: 4px;
    transition: all 0.2s ease;
  }

  .modal-close:hover {
    background: var(--bg-tertiary, #f3f4f6);
    color: var(--text-primary, #374151);
  }

  .modal-body {
    padding: 1.5rem;
    overflow-y: auto;
    flex: 1;
  }

  .preview-text {
    background: var(--bg-secondary, #f8fafc);
    border: 1px solid var(--border-secondary, #e2e8f0);
    border-radius: 6px;
    padding: 1rem;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.875rem;
    line-height: 1.5;
    white-space: pre-wrap;
    word-wrap: break-word;
    color: var(--text-primary, #374151);
    margin: 0;
  }

  .preview-content {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .preview-row {
    display: flex;
    align-items: flex-start;
    gap: 1rem;
    padding: 0.75rem;
    background: var(--bg-secondary, #f8fafc);
    border: 1px solid var(--border-secondary, #e2e8f0);
    border-radius: 6px;
  }

  .preview-key {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-accent, #667eea);
    min-width: 120px;
    flex-shrink: 0;
  }

  .preview-value {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.875rem;
    color: var(--text-primary, #374151);
    word-break: break-all;
    flex: 1;
  }

  .preview-empty {
    text-align: center;
    color: var(--text-secondary, #6b7280);
    font-style: italic;
    padding: 2rem;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    padding: 1.5rem;
    border-top: 1px solid var(--border-secondary, #e5e7eb);
  }

  .modal-footer .btn-secondary {
    background: var(--button-secondary, #f3f4f6);
    color: var(--text-primary, #374151);
    border: 1px solid var(--border-primary, #d1d5db);
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
    font-weight: 500;
    transition: all 0.2s ease;
  }

  .modal-footer .btn-secondary:hover {
    background: var(--button-secondary-hover, #e5e7eb);
  }

  .modal-footer .btn-primary {
    background: var(--button-primary, #667eea);
    color: white;
    border: 1px solid var(--button-primary, #667eea);
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
    font-weight: 500;
    transition: all 0.2s ease;
  }

  .modal-footer .btn-primary:hover {
    background: var(--button-primary-hover, #5a67d8);
    border-color: var(--button-primary-hover, #5a67d8);
  }

  /* Dark theme modal overrides */
  :global([data-theme="dark"]) .modal-content {
    background: var(--bg-primary);
    border: 1px solid var(--border-secondary);
  }

  :global([data-theme="dark"]) .modal-header {
    border-bottom-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .modal-header h3 {
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .modal-close {
    color: var(--text-secondary);
  }

  :global([data-theme="dark"]) .modal-close:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .preview-text {
    background: var(--bg-secondary);
    border-color: var(--border-secondary);
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .preview-row {
    background: var(--bg-secondary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .preview-key {
    color: var(--text-accent);
  }

  :global([data-theme="dark"]) .preview-value {
    color: var(--text-primary);
  }

  :global([data-theme="dark"]) .preview-empty {
    color: var(--text-secondary);
  }

  :global([data-theme="dark"]) .modal-footer {
    border-top-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .modal-footer .btn-secondary {
    background: var(--button-secondary);
    color: var(--text-primary);
    border-color: var(--border-secondary);
  }

  :global([data-theme="dark"]) .modal-footer .btn-secondary:hover {
    background: var(--button-secondary-hover);
  }
</style>