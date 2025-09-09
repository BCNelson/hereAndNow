# Claude Code Context: Here and Now Task Management

**Version**: 0.1.0 | **Updated**: 2025-09-08 | **Feature**: 001-build-an-application

## Project Overview

Context-aware task management system that filters tasks based on user's current location, calendar availability, and other contextual factors. Core principle: never show a task that can't be completed right now.

**Architecture**: Go library with SQLite database, multiple frontend integration methods.

## Technology Stack

- **Backend**: Go 1.21+, SQLite3 (WAL mode)
- **API**: REST with OpenAPI 3.0 spec
- **Database**: Embedded SQLite with full-text search
- **Testing**: Go testing package, testify for assertions
- **Deployment**: Single binary, cross-platform compilation
- **Performance**: Sub-100ms task filtering, 20 concurrent users

## Key Libraries & Dependencies

```go
// Core dependencies
"database/sql"
"github.com/mattn/go-sqlite3"  // SQLite driver
"github.com/google/uuid"       // UUID generation
"golang.org/x/crypto/argon2"   // Password hashing

// Planned additions
"github.com/gin-gonic/gin"     // HTTP router
"github.com/stretchr/testify"  // Testing assertions
"github.com/golang-migrate/migrate" // Database migrations
```

## Project Structure

```
/
├── cmd/hereandnow/          # CLI application
├── pkg/                     # Public Go packages (libraries)
│   ├── hereandnow/         # Core library
│   ├── models/             # Data models
│   ├── filters/            # Task filtering engine
│   └── sync/               # Calendar/location sync
├── internal/               # Private packages
│   ├── api/               # REST API handlers
│   ├── auth/              # Authentication
│   └── storage/           # Database layer
├── web/                   # Static web assets
├── tests/                 # All test files
│   ├── contract/          # API contract tests
│   ├── integration/       # Integration tests
│   └── unit/              # Unit tests
├── specs/                 # Feature specifications
├── scripts/               # Build/deployment scripts
└── docs/                  # Documentation
```

## Core Entities & Relationships

### Primary Entities
- **User**: Individual with authentication, preferences, timezone
- **Task**: Work unit with location requirements, time estimates, dependencies
- **Location**: Geographic position with radius for task completion
- **Context**: Current user state (location, available time, energy)
- **TaskList**: Container for tasks, supports sharing and hierarchy

### Key Relationships
- Users create/own Tasks and TaskLists
- Tasks can have multiple valid Locations
- Tasks depend on other Tasks (dependency graph)
- Context determines Task visibility through filtering engine

## Business Logic Patterns

### Task Filtering Engine
```go
type FilterRule interface {
    Apply(ctx Context, task Task) (visible bool, reason string)
}

// Core filtering rules:
// - LocationFilter: Must be at valid location within radius
// - TimeFilter: Must have enough available time
// - DependencyFilter: Prerequisites must be completed
// - EnergyFilter: Task energy requirement <= current energy
// - CalendarFilter: No conflicting calendar events
```

### Context Awareness
System continuously evaluates user context:
- GPS coordinates + named locations
- Calendar events → available time windows
- Social context (alone/with family/at work)
- Environmental data (weather, traffic)

## Database Schema Notes

- **SQLite WAL mode**: Enables concurrent reads during writes
- **FTS5 indexes**: Full-text search on task titles/descriptions
- **Spatial queries**: Haversine distance for location matching
- **Audit logging**: All filtering decisions tracked for transparency

## Testing Strategy

Following TDD with strict RED-GREEN-Refactor:

1. **Contract Tests**: API schema validation (must fail first)
2. **Integration Tests**: Real SQLite database, actual dependencies
3. **Unit Tests**: Individual function behavior
4. **Performance Tests**: Sub-100ms response time validation

## API Design Principles

- **RESTful**: Standard HTTP methods and status codes
- **Context-aware**: GET /tasks automatically filters by current context
- **Transparent**: Audit endpoints show why tasks are/aren't visible
- **Real-time**: Server-sent events for live updates
- **Versioned**: /api/v1/ with deprecation support

## Constitutional Compliance

### Simplicity ✓
- 3 projects max: core library, CLI, tests
- Direct SQLite access, no ORM
- Standard library preferred over frameworks

### Self-Hosted First ✓
- Embedded SQLite (no external database)
- Single binary deployment
- Optional cloud integrations with local fallback

### TDD Enforcement ✓
- Tests written before implementation
- Real dependencies (actual SQLite) in tests
- Git commits show failing tests first

## Recent Changes

### 2025-09-08: Initial Planning
- Created feature specification with 27 functional requirements
- Designed Go+SQLite architecture for self-hosting
- Defined REST API contracts with OpenAPI 3.0 spec
- Planned task filtering engine with rule-based transparency

## Commands for Development

```bash
# Build and test
make build          # Build all binaries
make test           # Run all tests
make test-contract  # API contract validation
make benchmark      # Performance benchmarks

# Database operations
make migrate-up     # Apply database migrations
make migrate-down   # Rollback migrations
make reset-db       # Fresh database (development only)

# Development server
make dev           # Hot-reload development server
make docker-dev    # Development in Docker

# Deployment
make release       # Cross-platform builds
make install       # Install to system PATH
```

## Key Design Decisions

1. **Go Library First**: Core logic in library, multiple frontend bindings
2. **SQLite Embedded**: Zero external dependencies for database
3. **Rule-based Filtering**: Transparent, debuggable task visibility
4. **Context Snapshots**: Discrete context evaluations for audit trail
5. **Event-driven Updates**: Real-time task list updates via SSE

## Performance Considerations

- **Query Optimization**: Prepared statements, strategic indexes
- **Caching Layer**: LRU cache for filtered task results
- **Batch Operations**: Minimize database round trips
- **Connection Pooling**: Efficient SQLite connection reuse

---

*This context file is automatically updated by the /plan command. Manual edits between markers are preserved.*