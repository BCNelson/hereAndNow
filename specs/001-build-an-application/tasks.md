# Tasks: Context-Aware Task Management System ("Here and Now")

**Input**: Design documents from `/specs/001-build-an-application/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/api-v1.yaml

## Execution Overview

This task list implements a Go-based context-aware task management system with SQLite database, following TDD principles. The system exposes core functionality as a Go library with multiple frontend integrations (CLI, API, mobile bindings).

**Tech Stack**: Go 1.21+, SQLite3 (mattn/go-sqlite3), Go testing framework
**Architecture**: Single project structure with core library, CLI, and comprehensive test suite
**Performance Target**: Sub-100ms task filtering, 20 concurrent users, <50MB memory

---

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Phase 3.1: Project Setup & Infrastructure

- [x] **T001** Create project directory structure per plan.md: `pkg/hereandnow/`, `pkg/models/`, `pkg/filters/`, `internal/storage/`, `internal/api/`, `cmd/hereandnow/`, `tests/{contract,integration,unit}/`
- [x] **T002** Initialize Go module with `go mod init github.com/bcnelson/hereAndNow` and add core dependencies: mattn/go-sqlite3, github.com/google/uuid, golang.org/x/crypto/argon2, github.com/stretchr/testify
- [x] **T003** [P] Create Makefile with build, test, lint, and development commands per quickstart.md requirements
- [x] **T004** [P] Configure golangci-lint with `.golangci.yml` for code quality enforcement
- [x] **T005** [P] Create initial database migration `migrations/001_initial_schema.sql` with all 12 entities from data-model.md

## Phase 3.2: Contract Tests (TDD - MUST FAIL FIRST) ⚠️

**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

### Authentication Contract Tests
- [x] **T006** [P] Contract test POST /auth/login in `tests/contract/auth_login_test.go` - validate login request/response schema
- [x] **T007** [P] Contract test POST /auth/logout in `tests/contract/auth_logout_test.go` - validate logout response

### User Management Contract Tests  
- [x] **T008** [P] Contract test GET /users/me in `tests/contract/users_me_test.go` - validate user profile schema
- [x] **T009** [P] Contract test PATCH /users/me in `tests/contract/users_update_test.go` - validate user update schema

### Task Management Contract Tests
- [x] **T010** [P] Contract test GET /tasks in `tests/contract/tasks_list_test.go` - validate filtered tasks response with context
- [x] **T011** [P] Contract test POST /tasks in `tests/contract/tasks_create_test.go` - validate task creation schema
- [x] **T012** [P] Contract test GET /tasks/{taskId} in `tests/contract/tasks_get_test.go` - validate task details schema
- [x] **T013** [P] Contract test PATCH /tasks/{taskId} in `tests/contract/tasks_update_test.go` - validate task update schema
- [x] **T014** [P] Contract test DELETE /tasks/{taskId} in `tests/contract/tasks_delete_test.go` - validate task deletion
- [x] **T015** [P] Contract test POST /tasks/{taskId}/assign in `tests/contract/remaining_endpoints_test.go` - validate task assignment schema
- [x] **T016** [P] Contract test POST /tasks/{taskId}/complete in `tests/contract/remaining_endpoints_test.go` - validate task completion
- [x] **T017** [P] Contract test GET /tasks/{taskId}/audit in `tests/contract/remaining_endpoints_test.go` - validate filtering audit schema
- [x] **T018** [P] Contract test POST /tasks/natural in `tests/contract/remaining_endpoints_test.go` - validate natural language parsing

### List Management Contract Tests
- [x] **T019** [P] Contract test GET /lists in `tests/contract/remaining_endpoints_test.go` - validate task lists schema
- [x] **T020** [P] Contract test POST /lists in `tests/contract/remaining_endpoints_test.go` - validate list creation schema
- [x] **T021** [P] Contract test GET /lists/{listId}/members in `tests/contract/remaining_endpoints_test.go` - validate list members schema
- [x] **T022** [P] Contract test POST /lists/{listId}/members in `tests/contract/remaining_endpoints_test.go` - validate member addition

### Context & Location Contract Tests
- [x] **T023** [P] Contract test GET /locations in `tests/contract/remaining_endpoints_test.go` - validate locations schema
- [x] **T024** [P] Contract test POST /locations in `tests/contract/remaining_endpoints_test.go` - validate location creation schema  
- [x] **T025** [P] Contract test GET /context in `tests/contract/context_get_test.go` - validate current context schema
- [x] **T026** [P] Contract test POST /context in `tests/contract/remaining_endpoints_test.go` - validate context update schema

### Integration & Analytics Contract Tests
- [x] **T027** [P] Contract test POST /calendar/sync in `tests/contract/remaining_endpoints_test.go` - validate calendar sync response
- [x] **T028** [P] Contract test GET /analytics in `tests/contract/remaining_endpoints_test.go` - validate analytics data schema
- [x] **T029** [P] Contract test GET /events (SSE) in `tests/contract/remaining_endpoints_test.go` - validate event stream format

## Phase 3.3: Data Models (After contract tests exist and fail)

### Core Entity Models
- [x] **T030** [P] User model in `pkg/models/user.go` - implement User struct with validation per data-model.md
- [x] **T031** [P] Task model in `pkg/models/task.go` - implement Task struct with TaskStatus enum and validation
- [x] **T032** [P] TaskList model in `pkg/models/task_list.go` - implement TaskList struct with hierarchy support
- [x] **T033** [P] Location model in `pkg/models/location.go` - implement Location struct with GPS coordinates validation
- [x] **T034** [P] Context model in `pkg/models/context.go` - implement Context struct for filtering decisions

### Relationship Models  
- [x] **T035** [P] TaskLocation model in `pkg/models/task_location.go` - implement many-to-many task-location relationship
- [x] **T036** [P] TaskDependency model in `pkg/models/task_dependency.go` - implement task dependencies with DependencyType enum
- [x] **T037** [P] CalendarEvent model in `pkg/models/calendar_event.go` - implement calendar integration model
- [x] **T038** [P] ListMember model in `pkg/models/list_member.go` - implement shared list membership with roles
- [x] **T039** [P] TaskAssignment model in `pkg/models/task_assignment.go` - implement task delegation with AssignmentStatus

### Audit & Analytics Models
- [x] **T040** [P] FilterAudit model in `pkg/models/filter_audit.go` - implement filtering transparency tracking
- [x] **T041** [P] Analytics model in `pkg/models/analytics.go` - implement productivity metrics aggregation

## Phase 3.4: Database Layer

- [x] **T042** Database connection manager in `internal/storage/db.go` - SQLite connection with WAL mode, migrations support
- [x] **T043** Migration runner in `internal/storage/migrate.go` - apply database schema migrations with rollback support
- [x] **T044** User repository in `internal/storage/user_repo.go` - CRUD operations for User entity with Argon2 password hashing
- [x] **T045** Task repository in `internal/storage/task_repo.go` - CRUD operations for Task entity with full-text search support
- [x] **T046** Location repository in `internal/storage/location_repo.go` - CRUD operations with spatial queries (Haversine distance)
- [x] **T047** Context repository in `internal/storage/context_repo.go` - context snapshots for audit trail

## Phase 3.5: Core Business Logic 

### Filtering Engine (Critical Path)
- [x] **T048** Filter interface in `pkg/filters/interface.go` - define FilterRule interface per research.md findings
- [x] **T049** Location filter in `pkg/filters/location.go` - implement location-based task filtering with radius calculation
- [x] **T050** Time filter in `pkg/filters/time.go` - implement time-based filtering using estimated minutes and calendar availability
- [x] **T051** Dependency filter in `pkg/filters/dependency.go` - implement dependency-based filtering with circular dependency detection
- [x] **T052** Priority filter in `pkg/filters/priority.go` - implement priority calculation with multiple factors
- [x] **T053** Filter engine in `pkg/filters/engine.go` - orchestrate all filters with transparent audit logging

### Core Services
- [x] **T054** Task service in `pkg/hereandnow/task_service.go` - core task management with filtering integration
- [x] **T055** Context service in `pkg/hereandnow/context_service.go` - context evaluation and management
- [x] **T056** Calendar sync service in `pkg/sync/calendar.go` - CalDAV integration using dolanor/caldav-go per research.md
- [x] **T057** Authentication service in `internal/auth/service.go` - JWT token management and user authentication

## Phase 3.6: API Implementation

### Authentication Endpoints
- [x] **T058** POST /auth/login handler in `internal/api/auth.go` - implement login with JWT token generation
- [x] **T059** POST /auth/logout handler in `internal/api/auth.go` - implement logout with token invalidation

### User Management Endpoints
- [x] **T060** GET /users/me handler in `internal/api/users.go` - get current user profile
- [x] **T061** PATCH /users/me handler in `internal/api/users.go` - update user profile with validation

### Task Management Endpoints  
- [x] **T062** GET /tasks handler in `internal/api/tasks.go` - get filtered tasks using context and filtering engine
- [x] **T063** POST /tasks handler in `internal/api/tasks.go` - create new task with location and dependency support
- [ ] **T064** GET /tasks/{taskId} handler in `internal/api/tasks.go` - get single task with relationships
- [ ] **T065** PATCH /tasks/{taskId} handler in `internal/api/tasks.go` - update task with status validation
- [ ] **T066** DELETE /tasks/{taskId} handler in `internal/api/tasks.go` - soft delete task with dependency validation
- [ ] **T067** POST /tasks/{taskId}/assign handler in `internal/api/tasks.go` - assign task to user with notifications
- [ ] **T068** POST /tasks/{taskId}/complete handler in `internal/api/tasks.go` - mark task complete with analytics
- [ ] **T069** GET /tasks/{taskId}/audit handler in `internal/api/tasks.go` - get filtering audit trail
- [ ] **T070** POST /tasks/natural handler in `internal/api/tasks.go` - create task from natural language input

### Context & Location Endpoints
- [ ] **T071** GET /context handler in `internal/api/context.go` - get current user context
- [ ] **T072** POST /context handler in `internal/api/context.go` - update user context (location, energy, etc.)
- [ ] **T073** GET /locations handler in `internal/api/locations.go` - get user's saved locations
- [ ] **T074** POST /locations handler in `internal/api/locations.go` - create new location with GPS validation

### List & Analytics Endpoints
- [ ] **T075** GET /lists handler in `internal/api/lists.go` - get user's task lists with sharing info
- [ ] **T076** POST /lists handler in `internal/api/lists.go` - create new task list
- [ ] **T077** GET /analytics handler in `internal/api/analytics.go` - get productivity analytics with date ranges
- [ ] **T078** GET /events (SSE) handler in `internal/api/events.go` - Server-Sent Events for real-time updates

## Phase 3.7: CLI Implementation ✅

### Core CLI Commands
- [x] **T079** [P] CLI main command in `cmd/hereandnow/main.go` - entry point with --help/--version/--format flags
- [x] **T080** [P] User commands in `cmd/hereandnow/user.go` - user create, list, update commands
- [x] **T081** [P] Task commands in `cmd/hereandnow/task.go` - task add, list, complete, assign commands per quickstart.md
- [x] **T082** [P] Location commands in `cmd/hereandnow/location.go` - location add, list, update commands
- [x] **T083** [P] Context commands in `cmd/hereandnow/context.go` - context show, update commands
- [x] **T084** [P] Server commands in `cmd/hereandnow/server.go` - serve, init, migrate commands

### CLI Infrastructure  
- [x] **T085** CLI configuration in `cmd/hereandnow/config.go` - config file management (.hereandnow/config.yaml)
- [x] **T086** CLI output formatting in `cmd/hereandnow/format.go` - JSON, table, and human-readable output

## Phase 3.8: Integration Tests (After core implementation)

### User Story Integration Tests
- [ ] **T087** [P] Scenario: Location-based filtering in `tests/integration/location_filtering_test.go` - verify tasks appear/disappear based on GPS location
- [ ] **T088** [P] Scenario: Time-based filtering in `tests/integration/time_filtering_test.go` - verify tasks filtered by available time windows
- [ ] **T089** [P] Scenario: Task dependencies in `tests/integration/dependencies_test.go` - verify dependent tasks hidden until prerequisites complete
- [ ] **T090** [P] Scenario: Shared task lists in `tests/integration/shared_lists_test.go` - verify real-time collaboration features
- [ ] **T091** [P] Scenario: Calendar integration in `tests/integration/calendar_sync_test.go` - verify calendar events affect task availability
- [ ] **T092** [P] Scenario: Natural language parsing in `tests/integration/natural_language_test.go` - verify "buy milk when at grocery store" parsing
- [ ] **T093** [P] Scenario: Task assignment workflow in `tests/integration/task_assignment_test.go` - verify delegation, acceptance, rejection flow

### System Integration Tests
- [ ] **T094** Database integration test in `tests/integration/database_test.go` - verify all repositories work with real SQLite
- [ ] **T095** API integration test in `tests/integration/api_test.go` - verify all endpoints work together with real database
- [ ] **T096** CLI integration test in `tests/integration/cli_test.go` - verify CLI commands work with real database and API

## Phase 3.9: Performance & Polish

### Performance & Optimization
- [ ] **T097** Performance test in `tests/performance/filtering_bench_test.go` - verify sub-100ms filtering with realistic data sets
- [ ] **T098** Concurrency test in `tests/performance/concurrent_test.go` - verify 20 concurrent users performance target
- [ ] **T099** Memory profiling in `tests/performance/memory_test.go` - verify <50MB memory footprint requirement

### Unit Tests & Documentation
- [ ] **T100** [P] Unit tests for filtering logic in `tests/unit/filters_test.go` - comprehensive filter rule testing
- [ ] **T101** [P] Unit tests for validation in `tests/unit/validation_test.go` - input validation edge cases
- [ ] **T102** [P] Unit tests for utilities in `tests/unit/utils_test.go` - UUID generation, password hashing, etc.
- [ ] **T103** [P] API documentation generation - update OpenAPI spec with examples and generate docs
- [ ] **T104** [P] Library documentation in `docs/library.md` - Go library usage examples and patterns

### Final Integration & Deployment
- [ ] **T105** Quickstart validation test in `tests/integration/quickstart_test.go` - automate all quickstart.md scenarios
- [ ] **T106** Cross-platform build test - verify compilation on Linux, macOS, Windows
- [ ] **T107** Production deployment test - verify single binary deployment with minimal dependencies

---

## Critical Dependencies

### Must Complete Before Implementation (TDD)
- **T006-T029** (All contract tests) → **T030-T107** (Any implementation)
- Contract tests MUST exist and MUST fail before writing any implementation code

### Core Implementation Flow
- **T042-T047** (Database layer) → **T048-T057** (Business logic)
- **T048-T053** (Filtering engine) → **T062** (GET /tasks endpoint) - Critical path
- **T054-T057** (Core services) → **T058-T078** (API endpoints)
- **T058-T078** (API implementation) → **T079-T086** (CLI implementation)

### Integration Dependencies  
- **T030-T086** (All implementation) → **T087-T096** (Integration tests)
- **T087-T096** (Integration tests) → **T097-T107** (Performance & polish)

## Parallel Execution Groups

### Setup Phase (Can run together)
```bash
# T001-T005: Project setup
Task: "Create project directory structure per plan.md"
Task: "Initialize Go module with dependencies"
Task: "Create Makefile with build, test, lint commands" 
Task: "Configure golangci-lint with .golangci.yml"
Task: "Create initial database migration with all 12 entities"
```

### Contract Tests Phase (Can run together)
```bash
# T006-T029: All contract tests (different files)
Task: "Contract test POST /auth/login in tests/contract/auth_login_test.go"
Task: "Contract test GET /tasks in tests/contract/tasks_list_test.go"
Task: "Contract test POST /tasks in tests/contract/tasks_create_test.go"
# ... all 24 contract test tasks can run in parallel
```

### Models Phase (Can run together)
```bash
# T030-T041: All model files (different files, no dependencies)
Task: "User model in pkg/models/user.go"
Task: "Task model in pkg/models/task.go" 
Task: "Location model in pkg/models/location.go"
# ... all 12 model tasks can run in parallel
```

### CLI Commands (Can run together after core services)
```bash
# T079-T086: CLI implementation (different files)
Task: "CLI main command in cmd/hereandnow/main.go"
Task: "User commands in cmd/hereandnow/user.go"
Task: "Task commands in cmd/hereandnow/task.go"
# ... all CLI tasks can run in parallel
```

### Integration Tests (Can run together after implementation)
```bash
# T087-T096: Integration tests (different test scenarios)
Task: "Location-based filtering integration test"
Task: "Time-based filtering integration test"
Task: "Task dependencies integration test"
# ... all integration tests can run in parallel
```

## Task Generation Validation ✅

- [x] All 15 API contract endpoints have corresponding contract tests (T006-T029)
- [x] All 12 entities from data-model.md have model creation tasks (T030-T041) 
- [x] All contract tests come before implementation tasks (Phase 3.2 → 3.3+)
- [x] Parallel tasks ([P]) operate on different files with no dependencies
- [x] Each task specifies exact file path for implementation
- [x] TDD principle enforced: tests must fail before implementation begins
- [x] Critical path identified: filtering engine blocks core functionality
- [x] Performance requirements covered: sub-100ms filtering, 20 concurrent users, <50MB memory
- [x] Integration tests cover all major user stories from spec.md
- [x] CLI implementation covers all quickstart.md scenarios

**Total Tasks**: 107 tasks across 9 phases
**Estimated Timeline**: 4-6 weeks for complete implementation
**Parallel Opportunities**: 65 tasks can run in parallel within their phases