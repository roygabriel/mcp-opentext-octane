# mcp-opentext-octane

> **Early Alpha** — This project is under active development and not yet production-ready. APIs, tool schemas, and behavior may change without notice. Use at your own risk.

A [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that exposes OpenText ALM Octane entities as tools for LLM consumption. Provides read and update operations for stories, features, epics, defects, tasks, releases, and work items within a single Octane workspace.

## Features

- 21 MCP tools (get, list, update) across 7 entity types
- Cookie-based session authentication with automatic re-auth on expiry
- Optimistic locking via automatic `version_stamp` handling on updates
- Octane query syntax support for filtering and searching
- Default field projections per entity type to keep responses focused
- Structured JSON logging with request tracing

## Requirements

- Go 1.25+
- Access to an OpenText ALM Octane instance
- API key credentials or username/password

## Installation

```bash
go install github.com/roygabriel/mcp-opentext-octane@latest
```

Or build from source:

```bash
git clone https://github.com/roygabriel/mcp-opentext-octane.git
cd mcp-opentext-octane
make build
```

## Configuration

Configuration is loaded from environment variables. An optional `.env` file is supported via [godotenv](https://github.com/joho/godotenv).

| Variable | Required | Description |
|----------|----------|-------------|
| `OCTANE_URL` | Yes | Base URL (e.g., `https://octane.example.com`) |
| `OCTANE_SHARED_SPACE_ID` | Yes | Shared space ID (numeric) |
| `OCTANE_WORKSPACE_ID` | Yes | Workspace ID (numeric) |
| `OCTANE_CLIENT_ID` | No* | API key client ID |
| `OCTANE_CLIENT_SECRET` | No* | API key client secret |
| `OCTANE_USERNAME` | No* | User login name |
| `OCTANE_PASSWORD` | No* | User password |
| `LOG_LEVEL` | No | Logging level: `DEBUG`, `INFO` (default), `WARN`, `ERROR` |

\* Either `CLIENT_ID` + `CLIENT_SECRET` or `USERNAME` + `PASSWORD` must be provided. API key authentication is preferred.

### Example `.env`

```env
OCTANE_URL=https://octane.example.com
OCTANE_SHARED_SPACE_ID=1001
OCTANE_WORKSPACE_ID=2001
OCTANE_CLIENT_ID=my_api_client_id@internal
OCTANE_CLIENT_SECRET=my_secret
```

## MCP Client Configuration

### Claude Desktop / Claude Code

Add to your MCP settings:

```json
{
  "mcpServers": {
    "octane": {
      "command": "/path/to/mcp-opentext-octane",
      "env": {
        "OCTANE_URL": "https://octane.example.com",
        "OCTANE_SHARED_SPACE_ID": "1001",
        "OCTANE_WORKSPACE_ID": "2001",
        "OCTANE_CLIENT_ID": "my_api_client_id@internal",
        "OCTANE_CLIENT_SECRET": "my_secret"
      }
    }
  }
}
```

## Tools

### Entity Types

| Entity | Get | List | Update |
|--------|-----|------|--------|
| User Story | `get_story` | `list_storys` | `update_story` |
| Feature | `get_feature` | `list_features` | `update_feature` |
| Epic | `get_epic` | `list_epics` | `update_epic` |
| Defect | `get_defect` | `list_defects` | `update_defect` |
| Task | `get_task` | `list_tasks` | `update_task` |
| Release | `get_release` | `list_releases` | `update_release` |
| Work Item | `get_work_item` | `list_work_items` | `update_work_item` |

### Get

Retrieves a single entity by ID.

**Parameters:**
- `id` (number, required) — Entity ID
- `fields` (string, optional) — Comma-separated field names. Omit to use entity-specific defaults

### List

Searches and lists entities with filtering, sorting, and pagination.

**Parameters:**
- `query` (string, optional) — Octane query filter (see [Query Syntax](#query-syntax))
- `fields` (string, optional) — Comma-separated field names. Omit to use entity-specific defaults
- `order_by` (string, optional) — Field name to sort by. Prefix with `-` for descending
- `limit` (number, default: 50) — Maximum results to return (1-200)
- `offset` (number, default: 0) — Number of results to skip for pagination

### Update

Updates fields on an existing entity. Version control (`version_stamp`) is handled automatically.

**Parameters:**
- `id` (number, required) — Entity ID
- `fields` (object, required) — Object of field names to new values

**Immutable fields** (cannot be updated): `id`, `type`, `subtype`, `creation_time`, `last_modified`, `version_stamp`, `workspace_id`, `logical_name`, `invested_hours`

## Query Syntax

The `query` parameter uses Octane's native query language:

| Pattern | Description | Example |
|---------|-------------|---------|
| `field='value'` | Exact match | `name='Login bug'` |
| `field='val*'` | Wildcard match | `name='Login*'` |
| `field={ref.value}` | Reference match | `phase={phase.new}` |
| `;` | AND | `phase={phase.new};priority={list_node.priority.high}` |
| `\|\|` | OR | `phase={phase.new}\|\|phase={phase.open}` |
| `field={null}` | Null check | `owner={null}` |
| `field>='date'` | Date comparison | `creation_time>='2024-01-01T00:00:00Z'` |

Maximum query length: 2,000 characters.

## Reference Fields

Reference fields are JSON objects with `id` and `type` properties. They are used when updating fields that point to other entities.

```json
// Set phase
{"phase": {"id": "phase.new", "type": "phase"}}

// Set priority
{"priority": {"id": "list_node.priority.critical", "type": "list_node"}}

// Assign owner (workspace user)
{"owner": {"id": 5001, "type": "workspace_user"}}

// Assign to sprint
{"sprint": {"id": 2001, "type": "sprint"}}

// Clear a field
{"sprint": null}
```

## Default Fields Per Entity

When the `fields` parameter is omitted, each entity type returns a curated set of default fields:

| Entity | Default Fields |
|--------|---------------|
| Story | id, name, phase, priority, story_points, owner, feature, release, milestone, sprint, team, blocked, blocked_reason, rank, item_origin, description, creation_time, last_modified |
| Feature | id, name, phase, priority, epic, release, milestone, rank, owner, description, creation_time, last_modified |
| Epic | id, name, phase, release, owner, description, creation_time, last_modified |
| Defect | id, name, phase, priority, story_points, owner, feature, release, milestone, sprint, team, blocked, blocked_reason, rank, item_origin, description, creation_time, last_modified |
| Task | id, name, phase, owner, backlog_item, sprint, team, estimated_hours, remaining_hours, invested_hours, blocked, blocked_reason, description, creation_time, last_modified |
| Release | id, name, start_date, end_date, creation_time, last_modified |
| Work Item | id, name, subtype, phase, priority, owner, release, sprint, team, creation_time, last_modified |

## Docker

Build and run with Docker:

```bash
# Build
docker build -t mcp-opentext-octane:latest .

# Or via Makefile
make docker
```

The image uses a [distroless](https://github.com/GoogleContainerTools/distroless) non-root base for minimal attack surface.

## Development

### Prerequisites

Install development tools:

```bash
make tools
```

This installs `golangci-lint` and `govulncheck`.

### Commands

| Command | Description |
|---------|-------------|
| `make all` | Run vet, lint, test, and build |
| `make build` | Build the binary |
| `make test` | Run tests with race detection and coverage |
| `make cover` | Generate HTML coverage report |
| `make vet` | Run `go vet` |
| `make lint` | Run `golangci-lint` |
| `make vuln` | Run `govulncheck` |
| `make run` | Build and run |
| `make docker` | Build Docker image |
| `make clean` | Remove binary and coverage files |

### Project Structure

```
mcp-opentext-octane/
├── config/
│   ├── config.go          # Environment-based configuration
│   └── config_test.go
├── octane/
│   ├── client.go          # HTTP client with auth and retry
│   ├── client_test.go
│   └── types.go           # Entity, EntityList, QueryParams, constants
├── tools/
│   ├── interfaces.go      # OctaneReader, OctaneWriter, OctaneService
│   ├── handlers.go        # Generic handler factories (get, list, update)
│   ├── handlers_test.go
│   ├── mock_test.go       # MockOctaneService for testing
│   ├── validate.go        # Input validation
│   └── validate_test.go
├── main.go                # Tool registration, middleware, server setup
├── middleware_test.go
├── Makefile
├── Dockerfile
├── .golangci.yml
└── sonar-project.properties
```

### Architecture

- **Generic handler pattern** — Three handler factories (`GetEntityHandler`, `ListEntitiesHandler`, `UpdateEntityHandler`) serve all 7 entity types, parameterized by entity type string
- **Dynamic entity model** — Octane has 4,500+ possible fields. Entities use `map[string]interface{}` with default field projections per type rather than typed structs
- **Cookie session auth** — Authenticates lazily on first request, stores `LWSSO_COOKIE_KEY` in a cookie jar, and retries once on 401 responses
- **Optimistic locking** — Updates perform a GET-then-PUT to obtain the current `version_stamp`, preventing stale-data conflicts
- **Testability** — External dependencies are behind interfaces (`OctaneReader`, `OctaneWriter`, `HTTPBackend`) for unit testing with mocks

### CI

GitHub Actions runs on push to `main`/`dev` and on pull requests:

- **Test** — `go vet`, tests with race detector, 80% coverage gate
- **Lint** — `golangci-lint` (errcheck, govet, staticcheck, unused, ineffassign, gosec, gocritic)
- **Vulnerability scan** — `govulncheck`
- **Build** — Depends on test and lint passing

Dependabot is configured for weekly updates to Go modules and GitHub Actions.

## License

See [LICENSE](LICENSE) for details.
