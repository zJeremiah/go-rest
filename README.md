# Go-Rest: Postman-like API Testing Tool

A modern, lightweight REST API testing tool built with Go and Svelte. Test your APIs with an intuitive web interface, organize requests into collections, manage environments, and use powerful template variables.

## 🚀 Features

### Core Functionality
- **HTTP Request Testing** - Support for GET, POST, PUT, DELETE, PATCH, HEAD, and OPTIONS methods
- **Request Collections** - Organize your API requests into named collections
- **Environment Management** - Switch between different environments (dev, staging, prod, etc.)
- **Template Variables** - Use `{{variable}}` syntax for dynamic request configuration
- **Response Display** - Beautiful syntax-highlighted JSON responses with copy functionality

### Advanced Features
- **Request Grouping** - Organize requests into logical groups
- **Search & Filter** - Search requests by name or URL, filter by groups
- **Response Variables** - Reference data from previous requests using `{{ "Request Name".json_key }}` syntax
- **JSON Object References** - Insert entire JSON objects/arrays as proper JSON (not escaped strings)
- **Deep Field Traversal** - Access nested JSON properties using dot notation (e.g., `address.geo.lat`)
- **Environment Variable References** - Reference system environment variables using `$ENV_VAR_NAME` syntax
- **Auto-save** - Automatic saving of request changes with immediate parameter saving
- **Request History** - View last response for each saved request
- **Keyboard Shortcuts** - Send requests with `Cmd+Enter` (Mac) or `Ctrl+Enter` (Windows/Linux)
- **Modal Previews** - Preview processed headers and body content with JSON-aware substitution
- **Request Duplication** - Quickly duplicate existing requests
- **Request Renaming** - In-place editing of request names with unique name validation

### Developer Experience
- **Modern UI** - Clean, responsive Svelte-based frontend with dark theme support
- **Smart Layout** - Optimized header bar with method, name, and URL input
- **Real-time Updates** - Live request/response cycle with loading states
- **Multiple Body Types** - Support for Text, JSON, and Form URL Encoded data
- **Error Handling** - Comprehensive error messages and status indicators
- **Data Persistence** - All data stored locally in JSON format with automatic migrations
- **CORS Support** - Built-in CORS handling for cross-origin requests

## 📦 Installation

### Prerequisites
- **Go 1.24.3+** - [Download Go](https://golang.org/dl/)
- **Node.js 18+** - [Download Node.js](https://nodejs.org/)
- **npm** - Comes with Node.js

### Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd go-rest
   ```

2. **Install frontend dependencies**
   ```bash
   cd frontend
   npm install
   ```

3. **Build the frontend**
   ```bash
   npm run build
   cd ..
   ```

4. **Install Go dependencies**
   ```bash
   go mod download
   ```

5. **Run the application**
   ```bash
   go run main.go
   ```

6. **Open your browser**
   Navigate to `http://localhost:8333`

## 🔧 Usage

### Creating Your First Request

1. **Add a New Request**
   - Click the "➕ Add Request" button
   - Enter a name for your request
   - Select or create a group

2. **Configure the Request**
   - Enter the URL (supports template variables like `{{host}}/api/users`)
   - Select HTTP method (GET, POST, etc.)
   - Add headers in the Headers tab
   - Add query parameters in the Params tab
   - Add request body for POST/PUT requests (supports Text, JSON, and Form data)

3. **Send the Request**
   - Click "🚀 Send"
   - View the response with syntax highlighting
   - Copy response data with the 📋 Copy button

### Managing Environments

Environments allow you to switch between different sets of variables (e.g., dev vs production URLs).

1. **Create an Environment**
   - Click the "⚙️" settings icon
   - Click "Create Environment"
   - Enter environment name and variables

2. **Switch Environments**
   - Use the environment dropdown to switch between environments
   - All `{{variable}}` placeholders will be replaced with environment values

3. **Environment Variables**
   - Format: `{{variable_name}}`
   - Example: `{{host}}/api/users` where `host` might be `https://api.example.com`
   
4. **Environment Variable References**
   - Reference system environment variables by prefixing variable values with `$`
   - Format: Set variable value to `$ENV_VAR_NAME`
   - Example: Set `api_key` variable value to `$API_KEY` to reference the system's `API_KEY` environment variable  
   - Use case: `Authorization: Bearer {{api_key}}` where `api_key` value is `$SECRET_TOKEN`
   - Benefits: Keep sensitive data out of configuration files, use system environment for dynamic values

### Using Response Variables

Access data from previous request responses to create dynamic request chains:

1. **Basic Syntax**
   - Format: `{{ "Request Name".json_key }}`
   - Example: `{{ "Auth".access_token }}` extracts `access_token` from the "Auth" request response

2. **Deep Field Traversal**
   - Use dot notation to access nested JSON properties
   - Format: `{{ "Request Name".parent.child.property }}`
   - Example: `{{ "User Profile".address.geo.lat }}` extracts latitude from nested address object

3. **JSON Object References**
   - **Primitive Values**: `{{ "Request".user.name }}` → Returns `"John Doe"` (string)
   - **JSON Objects**: `{{ "Request".user.address }}` → Returns `{"city": "New York", "zip": "10001"}` (actual JSON object)
   - **Array Values**: `{{ "Request".user.tags }}` → Returns `["admin", "user"]` (actual JSON array)

4. **Use Cases**
   - **Authentication tokens**: `{{ "Login".token }}`
   - **Dynamic IDs**: `{{ "Create User".user_id }}`
   - **Nested primitive values**: `{{ "User Profile".data.email }}`
   - **Nested coordinates**: `{{ "Location".address.geo.lat }}`
   - **Complete objects**: `{{ "API Response".user }}` (inserts entire user object)
   - **Sub-objects**: `{{ "API Response".user.preferences }}` (inserts preferences object)

5. **Smart JSON Handling**
   - When referencing objects, they are inserted as proper JSON (not escaped strings)
   - Preview functionality shows actual JSON structure
   - Compatible with all request body types (Text, JSON, Form)

6. **Request Names**
   - Must be case-sensitive exact matches
   - Use quotes to handle names with spaces
   - Names must be unique across all requests

### Request Organization

- **Groups**: Organize requests into logical groups (Authentication, Users, Orders, etc.)
- **Collections**: All requests are automatically saved to your collection
- **Search**: Use the search bar to quickly find requests by name or URL
- **Filtering**: Filter requests by group using the group dropdown
- **Combined Filtering**: Use group filter and search together for precise request finding

### Keyboard Shortcuts

- **Send Request**: `Cmd+Enter` (Mac) or `Ctrl+Enter` (Windows/Linux)
- **Quick Navigation**: Use search to jump to specific requests

## 🔧 Configuration

### Environment Variables

The application supports the following environment variables:

- `PORT` - Server port (default: 8333)

### Environment Variable References

You can reference system environment variables in your template variables by prefixing the value with `$`:

**Setup Example:**
1. Set a system environment variable:
   ```bash
   export API_KEY="your-secret-api-key"
   export BASE_URL="https://api.production.com"
   ```

2. Create template variables in your environment:
   - Variable name: `auth_token`, Value: `$API_KEY`
   - Variable name: `host`, Value: `$BASE_URL`

3. Use in requests:
   - URL: `{{host}}/users`
   - Header: `Authorization: Bearer {{auth_token}}`

**Benefits:**
- Keep sensitive data out of `saved_requests.json`
- Use different values per deployment environment
- Dynamic configuration without code changes

### Data Storage

All data is stored locally in `saved_requests.json` in the project root. This file contains:
- Request definitions with separate body types (Text, JSON, Form URL Encoded)
- Response history for response variable references
- Environment configurations with template variables (including `$ENV_VAR_NAME` references)
- Group definitions for request organization
- Application settings and UI preferences

**Note**: Add `saved_requests.json` to your `.gitignore` if it contains sensitive data. Environment variable references (`$VAR_NAME`) are stored as references only - actual values come from your system environment.

## 🏗️ Development

### Project Structure

```
go-rest/
├── main.go                 # Go server and API endpoints
├── go.mod                  # Go dependencies
├── saved_requests.json     # Data storage (created automatically)
├── frontend/              # Svelte frontend
│   ├── src/
│   │   ├── lib/           # Svelte components
│   │   │   ├── RequestForm.svelte
│   │   │   └── ResponseDisplay.svelte
│   │   └── routes/        # SvelteKit routes
│   │       └── +page.svelte
│   ├── package.json
│   └── dist/             # Built frontend (created by npm run build)
└── README.md
```

### Backend API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/proxy` | Proxy HTTP requests to external APIs |
| GET | `/api/requests` | Get all saved requests |
| POST | `/api/requests/save` | Save a new request |
| PUT | `/api/requests/update` | Update an existing request |
| DELETE | `/api/requests/delete` | Delete a request |
| POST | `/api/requests/duplicate` | Duplicate a request |
| GET | `/api/environments` | Get all environments |
| POST | `/api/environments` | Create a new environment |
| PUT | `/api/environments/{id}` | Update an environment |
| DELETE | `/api/environments/{id}` | Delete an environment |
| GET | `/api/groups` | Get all groups |
| POST | `/api/groups` | Create a new group |

### Frontend Development

1. **Start development server**
   ```bash
   cd frontend
   npm run dev
   ```

2. **Build for production**
   ```bash
   npm run build
   ```

3. **Preview production build**
   ```bash
   npm run preview
   ```

### Backend Development

The Go server serves the built frontend from `frontend/dist/` and provides API endpoints for request management.

**Key Features:**
- Chi router for HTTP routing
- JSON-based data storage
- CORS middleware
- Request logging
- Template variable processing
- Response parsing and formatting

## 🔒 Security Considerations

- The application runs a local server that can make requests to any URL
- Be cautious when sharing `saved_requests.json` as it may contain sensitive data
- **Use Environment Variable References** - Store sensitive data (API keys, tokens) in system environment variables using `$ENV_VAR_NAME` syntax instead of hardcoding values
- Environment variable references keep secrets out of configuration files that might be committed to version control
- The tool is designed for development/testing, not production use

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and formatting (`go fmt`)
- Use meaningful commit messages
- Test your changes thoroughly
- Update documentation as needed

## 📝 License

This project is open source. See the LICENSE file for details.

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/) and [Svelte](https://svelte.dev/)
- Uses [Chi](https://github.com/go-chi/chi) for HTTP routing
- Inspired by [Postman](https://postman.com/) and similar API testing tools

## 🐛 Troubleshooting

### Common Issues

**Frontend not loading**
- Ensure you've built the frontend: `cd frontend && npm run build`
- Check that `frontend/dist/` directory exists

**Requests failing**
- Check CORS settings on the target API
- Verify URL format and template variables
- Check the browser console for errors

**Data not persisting**
- Ensure the application has write permissions in the project directory
- Check if `saved_requests.json` is being created

**Port already in use**
- Change the port: `PORT=3000 go run main.go`
- Kill existing processes using the port

**Environment variables not resolving**
- Ensure environment variables are set: `echo $YOUR_VAR_NAME`
- Restart the application after setting new environment variables
- Check variable names are exact matches (case-sensitive)
- Variables showing as `$VAR_NAME` instead of values means the environment variable is not set

---

**Happy API Testing!** 🚀