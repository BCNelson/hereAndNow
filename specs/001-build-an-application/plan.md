# Implementation Plan: Context-Aware Task Management System ("Here and Now")

**Branch**: `001-build-an-application` | **Date**: 2025-09-08 | **Spec**: `/specs/001-build-an-application/spec.md`
**Input**: Feature specification from `/specs/001-build-an-application/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
4. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
5. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, or `GEMINI.md` for Gemini CLI).
6. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
7. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
8. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Building a context-aware task management system that filters tasks based on user's current location, calendar availability, and other contextual factors. The system will use a Go backend with SQLite database for the core logic, exposed as a library that can be compiled and integrated with multiple frontend implementations. This approach ensures maximum portability while maintaining the self-hosted, privacy-first philosophy.

## Technical Context
**Language/Version**: Go 1.21+  
**Primary Dependencies**: SQLite3, Go standard library, potential UI bindings per frontend  
**Storage**: SQLite database (embedded, self-hosted friendly)  
**Testing**: Go testing package, testify for assertions  
**Target Platform**: Cross-platform (Linux, macOS, Windows) via Go compilation
**Project Type**: single (Go library with multiple frontend adapters)  
**Performance Goals**: Sub-100ms query response for task filtering, support 20 concurrent users  
**Constraints**: <50MB memory footprint, offline-capable, must run on Raspberry Pi 4  
**Scale/Scope**: ~20 concurrent users (home/family scale), extensible to small teams

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Simplicity**:
- Projects: 3 (core library, CLI, tests)
- Using framework directly? Yes - Go standard library, SQLite directly
- Single data model? Yes - shared structs across library
- Avoiding patterns? Yes - direct DB access, no unnecessary abstractions

**Architecture**:
- EVERY feature as library? Yes - core logic in Go library
- Libraries listed:
  - hereandnow: Core task management and filtering logic
  - hereandnow-sync: Calendar and location synchronization
  - hereandnow-ml: Future ML-based recommendations (Phase 2)
- CLI per library: hereandnow CLI with --help/--version/--format=json
- Library docs: llms.txt format planned? Yes

**Testing (NON-NEGOTIABLE)**:
- RED-GREEN-Refactor cycle enforced? Yes - tests written first
- Git commits show tests before implementation? Yes
- Order: Contract→Integration→E2E→Unit strictly followed? Yes
- Real dependencies used? Yes - real SQLite DB in tests
- Integration tests for: new libraries, contract changes, shared schemas? Yes
- FORBIDDEN: Implementation before test, skipping RED phase - Understood

**Observability**:
- Structured logging included? Yes - using slog (structured logging)
- Frontend logs → backend? Yes - unified log aggregation planned
- Error context sufficient? Yes - context propagation via context.Context

**Versioning**:
- Version number assigned? 0.1.0 (initial release)
- BUILD increments on every change? Yes - automated via CI
- Breaking changes handled? Yes - SQL migrations, API versioning

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
# Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure]
```

**Structure Decision**: Option 1 (Single project) - Go library with frontends as separate consumers

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `/scripts/update-agent-context.sh [claude|gemini|copilot]` for your AI assistant
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- API contract tests → one test per endpoint group (10 tasks)
- Data model → entity creation + migration tasks (15 tasks)
- Core libraries → filtering engine + sync components (12 tasks) 
- CLI implementation → command handlers + validation (8 tasks)
- Integration tests → user story validation scenarios (10 tasks)
- Performance/deployment tasks (5 tasks)

**Ordering Strategy**:
- TDD order enforced: ALL tests written before any implementation
- Dependency order: Database → Models → Services → API → CLI
- Infrastructure first: Migrations, auth, logging setup
- Mark [P] for parallel execution within same dependency level
- Critical path: Core filtering logic blocks most features

**Task Categories by Phase**:
1. **Setup & Infrastructure** (5 tasks): Database, migrations, logging
2. **Contract Tests** (10 tasks): API endpoint validation, must fail initially
3. **Data Layer** (8 tasks): Models, storage interfaces, migrations  
4. **Core Business Logic** (12 tasks): Filtering engine, context evaluation
5. **API Implementation** (10 tasks): REST handlers, authentication
6. **CLI Implementation** (8 tasks): Command handlers, user interface
7. **Integration & E2E** (7 tasks): User story validation, performance tests

**Estimated Output**: 60 numbered, ordered tasks in tasks.md

**Parallel Execution Groups**:
- Database + Models + Auth (independent setup)
- Individual API endpoints (after contracts exist)
- CLI commands (after core library exists)
- Integration tests (after implementation complete)

**Critical Dependencies**:
- All contract tests must exist before any implementation
- Core filtering engine blocks task visibility features
- Authentication blocks all API endpoints
- Database migrations block all data operations

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented (none required)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*