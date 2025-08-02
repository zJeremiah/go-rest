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
- **Auto-save** - Automatic saving of request changes with manual save option
- **Request History** - View last response for each saved request
- **Word Wrap** - Toggle word wrapping for better response readability
- **Request Duplication** - Quickly duplicate existing requests
- **Request Renaming** - In-place editing of request names

### Developer Experience
- **Modern UI** - Clean, responsive Svelte-based frontend
- **Real-time Updates** - Live request/response cycle with loading states
- **Error Handling** - Comprehensive error messages and status indicators
- **Data Persistence** - All data stored locally in JSON format
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
   Navigate to `http://localhost:8080`

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
   - Click "📤 Send Request"
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

### Request Organization

- **Groups**: Organize requests into logical groups (Authentication, Users, Orders, etc.)
- **Collections**: All requests are automatically saved to your collection
- **Search**: Use the search bar to quickly find requests
- **Filtering**: Filter requests by group using the group dropdown

## 🔧 Configuration

### Environment Variables

The application supports the following environment variables:

- `PORT` - Server port (default: 8080)

### Data Storage

All data is stored locally in `saved_requests.json` in the project root. This file contains:
- Request definitions
- Response history
- Environment configurations
- Group definitions
- Application settings

**Note**: Add `saved_requests.json` to your `.gitignore` if it contains sensitive data.

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
- Consider using environment variables for sensitive configuration
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

---

**Happy API Testing!** 🚀