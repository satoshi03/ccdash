# CCDash

CCDash is a web application that monitors the execution status of Claude Code and performs task scheduling.

## 📋 Supported Platforms

CCDash supports the following platforms:

| Platform | Architecture | Status |
|----------|-------------|---------|
| **macOS** | Intel (x64) | ✅ Supported |
| **macOS** | Apple Silicon (ARM64) | ✅ Supported |
| **Linux** | x64 | ✅ Supported |
| **Windows** | x64 | ✅ Supported |

> **Note**: Linux ARM64 is not currently supported due to build complexity with DuckDB CGO dependencies.

## Quickstart

```bash
# Run instantly using NPX
npx ccdash

# Or, global installation
npm install -g ccdash
ccdash

# Start with custom frontend port
ccdash --frontend-port 3001

# Start with custom frontend URL
ccdash --frontend-url https://app.example.com
```

After installation, access the following URLs in your browser:

  - **Frontend**: http://localhost:3000 (or your custom port/URL)
  - **Backend API**: http://localhost:6060 (fixed port)

## Overview

![CCDash Screenshot](assets/images/ccdash_top_1.png)

CCDash is a tool that visualizes Claude Code usage and helps with efficient task management. It supports monitoring token usage, managing sessions, and will support automatic task scheduling in the future.

## Features

### Currently Implemented

#### Monitoring Features

  - **Token Usage Monitoring**

      - Calculates usage within a 5-hour window
      - Displays usage rate (percentage of limit)
      - Supports different plan limits (Pro/Max5/Max20)
      - Displays reset time

  - **Session Management**

      - Displays a list of sessions
      - Categorization by project
      - Session details (token usage, message count, status)
      - Calculation of execution time

  - **Data Synchronization**

      - Automatic parsing of Claude Code JSONL log files
      - Automatic synchronization to the database

#### UI Features

  - Responsive design
  - Dark mode support
  - Real-time data updates
  - Error handling

### Future Implementations

  - **Task Scheduling**

      - Automatic execution after token reset
      - Manual execution, cancellation, and deletion functions
      - Priority setting

  - **Extended Features**

      - Usage statistics and analysis
      - Export functionality
      - Notification feature

## Architecture

```
ccdash/
├── backend/          # Go/Gin Backend
│   ├── cmd/server/   # Application entry point
│   ├── internal/     # Internal packages
│   │   ├── handlers/ # HTTP Handlers
│   │   ├── services/ # Business Logic
│   │   ├── models/   # Data Models
│   │   └── database/ # Database Connection
│   └── configs/      # Configuration files
├── frontend/         # Next.js Frontend
│   ├── app/          # App Router
│   ├── components/   # UI Components
│   ├── lib/          # Utilities
│   └── hooks/        # Custom Hooks
└── docs/             # Documentation
```

## Technology Stack

### Backend

  - **Go 1.21+**: Programming Language
  - **Gin**: Web Framework
  - **DuckDB**: Database
  - **CORS**: Cross-Origin Resource Sharing

### Frontend

  - **Next.js 15**: React Framework
  - **TypeScript**: Type Safety
  - **Tailwind CSS**: Styling
  - **shadcn/ui**: UI Components
  - **Lucide React**: Icons

### Development & Operations

  - **Git**: Version Control
  - **npm/pnpm**: Package Management
  - **Go Modules**: Dependency Management

## Setup

### Prerequisites

  - Go 1.21 or higher
  - Node.js 18 or higher
  - npm or pnpm

### Installation

1.  **Clone the repository**

    ```bash
    git clone <repository-url>
    cd ccdash
    ```

2.  **Backend setup**

    ```bash
    cd backend
    go mod download
    ```

3.  **Frontend setup**

    ```bash
    cd frontend
    npm install --legacy-peer-deps
    # Or
    pnpm install --legacy-peer-deps
    ```

## Commands

CCDash offers the following commands:

```bash
# Start the application (production mode)
npx ccdash
npx ccdash start

# Start with custom frontend port
npx ccdash --frontend-port 3001
npx ccdash -fp 3001

# Start with custom frontend URL
npx ccdash --frontend-url https://app.example.com
npx ccdash -fu https://app.example.com

# Start in development mode (with hot-reloading)
npx ccdash dev
npx ccdash dev --frontend-port 3001

# Build the application
npx ccdash build

# Display help
npx ccdash help

# Display version
npx ccdash version
```

### Command Line Options

- `--frontend-port, -fp`: Frontend server port (default: 3000)  
- `--frontend-url, -fu`: Frontend server URL (overrides frontend-port)
- `--help, -h`: Show help message

**Note**: Backend port is fixed at 6060 for npm package distribution.

### Prerequisites

  - **Node.js**: 18.0.0 or higher
  - **Go**: 1.21 or higher (required for building the backend)

## Usage

### Starting in Production Environment

```bash
npx ccdash
```

### Starting in Development Environment

```bash
# Start in development mode (recommended)
npx ccdash dev

# Or manually start each service
```

1.  **Start the backend server**

    ```bash
    cd backend
    go run cmd/server/main.go
    ```

    The server will start at `http://localhost:6060`.

2.  **Start the frontend server**

    ```bash
    cd frontend
    npm run dev
    # Or
    pnpm dev
    ```

    The application will start at `http://localhost:3000`.

### Starting in Production Environment

1.  **Build the backend**

    ```bash
    cd backend
    go build -o bin/server cmd/server/main.go
    ./bin/server
    ```

2.  **Build the frontend**

    ```bash
    cd frontend
    npm run build
    npm start
    ```

## API Specification

### Endpoints

  - `GET /api/v1/health` - Health check
  - `GET /api/token-usage` - Get token usage
  - `GET /api/claude/sessions/recent` - List of recent sessions
  - `GET /api/claude/available-tokens` - Available tokens
  - `GET /api/costs/current-month` - Monthly cost (planned)
  - `GET /api/tasks` - List of tasks (planned)
  - `POST /api/sync-logs` - Execute log synchronization

### Data Format

Example token usage response:

```json
{
  "total_tokens": 4250,
  "input_tokens": 2500,
  "output_tokens": 1750,
  "usage_limit": 7000,
  "usage_rate": 0.607,
  "window_start": "2024-01-01T10:00:00Z",
  "window_end": "2024-01-01T15:00:00Z",
  "active_sessions": 2
}
```

## Configuration

### Environment Variables

#### Backend

  - `GIN_MODE`: Gin operation mode (development/release)
  - `CCDASH_DB_PATH`: Path to the database file (default: `~/.ccdash/ccdash.db`)
  - `PORT`: Backend server port (fixed: 6060)
  - `FRONTEND_URL`: Frontend URL for CORS configuration (default: `http://localhost:3000`)

#### Frontend

  - `NEXT_PUBLIC_API_URL`: Backend API URL (default: `http://localhost:6060/api`)

### Claude Code Configuration

CCDash parses JSONL log files generated by Claude Code.
Log file location: `~/.claude/projects/{project-name}/{session-id}.jsonl`

### Custom Frontend Configuration Examples

#### Using Custom Frontend Port
```bash
# Frontend on custom port  
npx ccdash --frontend-port 8080
npx ccdash -fp 3001
```

#### Using Custom Frontend URL
```bash
# Frontend on custom URL
npx ccdash --frontend-url https://app.myserver.com
npx ccdash -fu http://app.local:3000
```

#### Development with Custom Frontend Configuration
```bash
# Development mode with custom frontend port
npx ccdash dev --frontend-port 3001

# Development with custom frontend URL
npx ccdash dev --frontend-url http://dev.app.local:3000
```

**Note**: Backend runs on fixed port 6060 and cannot be customized in npm package mode.

### CORS Configuration and Nginx Recommendation

When using custom frontend URLs (especially with different domains or ports), you may encounter CORS (Cross-Origin Resource Sharing) issues. For production deployments with custom domains, **we strongly recommend using nginx as a reverse proxy** instead of directly configuring frontend URLs.

#### Why nginx is recommended:
- Eliminates CORS issues by serving both frontend and backend from the same origin
- Better performance and security
- SSL termination and load balancing capabilities
- Professional production deployment approach

For detailed nginx configuration, see [nginx/README.md](nginx/README.md).

#### Alternative: Using custom frontend URLs (may cause CORS issues)

If you still prefer to use custom frontend URLs directly:

```bash
# These may cause CORS issues in production
npx ccdash --frontend-url https://app.yourdomain.com
npx ccdash --frontend-port 8080
```

**Important notes when using custom frontend URLs:**
- Ensure the backend server's CORS settings allow your frontend domain
- Consider using localhost or same-origin setups to avoid CORS issues
- Test thoroughly in your target environment before production deployment
- Monitor browser console for CORS-related error messages

## Troubleshooting

### Common Issues

1.  **Database Lock Error**

    ```bash
    rm -f ~/.ccdash/ccdash.db*
    ```

2.  **Frontend Dependency Error**

    ```bash
    cd frontend
    npm install --legacy-peer-deps
    ```

3.  **CORS Error**

    When using custom frontend URLs, you may encounter CORS errors. Solutions:
    
    ```bash
    # Option 1: Use nginx reverse proxy (recommended)
    # See nginx/README.md for configuration details
    
    # Option 2: Use localhost with custom ports
    npx ccdash --frontend-port 8080  # Same origin, different port
    
    # Option 3: Check browser console for specific CORS errors
    # Ensure backend allows your frontend domain in CORS settings
    ```
    
    **Recommended solution**: Use nginx reverse proxy to serve both frontend and backend from the same origin.

### Logs

  - Backend logs are output to standard output
  - Frontend logs can be viewed in the browser's developer tools

## Contributing

Pull requests and issues are welcome.

### Development Workflow

1.  Create an issue
2.  Create a feature branch
3.  Implement changes
4.  Run tests
5.  Create a pull request

## License

MIT License

## References

  - [Claude Code Documentation](https://docs.anthropic.com/claude/docs)
  - [Go Documentation](https://golang.org/doc/)
  - [Next.js Documentation](https://nextjs.org/docs)
  - [DuckDB Documentation](https://duckdb.org/docs/)
