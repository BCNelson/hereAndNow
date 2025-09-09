---
allowed-tools: Read, Write, Edit, MultiEdit, Bash, Glob, Grep, TodoWrite
argument-hint: [spec-name] [mode] [task-id|phase]
description: Execute tasks from a tasks.md file systematically, tracking progress and handling dependencies
---

Execute tasks from a tasks.md file systematically, tracking progress and handling dependencies.

This is the fourth step in the Spec-Driven Development lifecycle.

Given the context provided as an argument, do this:

**Usage**: `/implement [SPEC_NAME] [MODE] [TASK_ID]`
- `$1` = SPEC_NAME: Feature directory (e.g., "001-build-an-application"). If empty, auto-detect from `/specs/` directory
- `$2` = MODE: Execution mode (next|parallel|resume|task|phase|validate). Default: "next"
- `$3` = TASK_ID or PHASE: For mode=task use task ID (e.g., "T042"), for mode=phase use phase name (e.g., "3.4")

1. **Load and validate task file**:
   - Parse SPEC_NAME from $1 or auto-detect from `specs/` directory if $1 is empty
   - If multiple specs available and $1 empty, prompt user to select one
   - Read the tasks.md file from `specs/$1/tasks.md`  
   - Parse task structure: `[ID] [P?] Description` with checkbox status `[ ]` or `[x]`
   - Validate task dependencies and identify parallel execution groups
   - Report total tasks, completed tasks, and current phase

2. **Assess current state**:
   - Count completed tasks (marked with `[x]`)
   - Identify next available tasks that can be started (dependencies met)
   - List any blocked tasks and their blocking dependencies
   - Show parallel execution opportunities (tasks marked with `[P]`)

3. **Execute next task(s)** based on $2 (MODE):
   - **next** (default): Execute the next sequential task in dependency order
   - **parallel**: Execute multiple `[P]` tasks that can run concurrently
   - **resume**: Continue from where work was previously paused
   - **task**: Execute specific task by ID from $3 (e.g., T042)
   - **phase**: Execute all tasks in phase from $3 (e.g., "3.4")
   - **validate**: Verify current state without executing tasks

4. **Task execution workflow**:
   ```
   For each task being executed:
   a) Read the task description and extract file paths
   b) Check if any prerequisite files/dependencies exist
   c) Create or edit the specified files following the task requirements
   d) Run relevant tests to validate the implementation
   e) Update the task status to [x] in the tasks.md file
   f) Commit changes with descriptive message: "Implement [TaskID]: [Description]"
   ```

5. **State management**:
   - Always update task completion status immediately after successful implementation
   - Create checkpoint commits for major milestones (end of phases)
   - Save progress comments in tasks.md for complex tasks that are partially complete
   - Handle failures gracefully - mark tasks as blocked if dependencies fail

6. **TDD enforcement** (Critical for contract tests T006-T029):
   - Contract tests MUST be written first and MUST fail before implementation
   - Verify test failure before proceeding to implementation
   - Never mark contract tests complete until they fail as expected
   - Run all tests after each implementation task to ensure no regressions

7. **Quality checks after each task**:
   - Run `make lint` if available, otherwise `golangci-lint run`
   - Run `make test` if available, otherwise `go test ./...`
   - For API tasks: validate against OpenAPI spec if present
   - For performance tasks: run benchmarks and validate requirements

8. **Progress reporting**:
   - Show completion percentage: `[45/107] 42% complete - Phase 3.4: Database Layer`
   - List recently completed tasks (last 5)
   - Show next 3-5 available tasks with time estimates
   - Identify blocking issues and suggest resolution paths

9. **Error handling and recovery**:
    - If a task fails: mark as blocked, identify cause, suggest fixes
    - If tests fail: show test output, identify failing tests, suggest solutions
    - If dependencies missing: install required packages, update go.mod
    - If file conflicts: show diff, suggest resolution strategy
    - Create recovery checkpoints before major phases

11. **Parallel execution coordination**:
    ```bash
    # Example parallel execution for Phase 3.2 (Contract Tests)
    # Execute T006-T029 concurrently since they operate on different files
    Task("Contract test POST /auth/login") & 
    Task("Contract test GET /tasks") &
    Task("Contract test POST /tasks") &
    # Wait for all to complete before proceeding to Phase 3.3
    ```

12. **Smart dependency resolution**:
    - Automatically detect when blocked tasks become available
    - Suggest optimal execution order within parallel groups
    - Handle cross-phase dependencies (e.g., all contract tests before models)
    - Warn about circular dependencies or missing prerequisites

13. **Integration with development workflow**:
    - Auto-create branches for major phases: `feature/phase-3.4-database`
    - Commit frequently with standardized messages
    - Tag major milestones: `v0.1.0-phase3.4-complete`
    - Generate progress reports for project tracking

14. **File and directory management**:
    - Create directory structure as needed per tasks
    - Backup important files before major changes
    - Maintain clean working directory (no uncommitted changes between tasks)
    - Follow project conventions from CLAUDE.md and existing codebase

15. **Context preservation**:
    - Save current task context to `.claude/state/current-task.json`
    - Track time spent on each task for estimation improvement
    - Log decision rationale for complex implementation choices
    - Maintain audit trail of what was implemented and why

16. **Multi-spec support**:
    - Handle multiple feature specifications in parallel
    - Maintain separate state files per spec: `.claude/state/$1-task.json`
    - Support switching between active specifications
    - Cross-reference dependencies between different specs if needed

**Positional Arguments**: 
- `$1` = SPEC_NAME: Feature directory name (e.g., "001-build-an-application")
- `$2` = MODE: Execution mode (next|parallel|resume|task|phase|validate)
- `$3` = TASK_ID or PHASE: Task identifier or phase name depending on mode

**Output Format**: Always show current progress, next available tasks, and any blocking issues. Update the tasks.md file immediately after each successful task completion. Provide clear status on what was accomplished and what comes next.

**Critical Success Factors**:
- Follow TDD strictly: tests first, implementation second
- Update task status immediately after completion
- Respect dependencies - never skip prerequisite tasks
- Maintain code quality with linting and testing after each task
- Commit progress frequently to enable easy recovery

The implementation should be immediately resumable - anyone should be able to run `/implement --mode=resume` and continue from exactly where work was paused.