-- Initial schema for Here and Now Task Management System
-- Date: 2025-09-09
-- Version: 1.0.0

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

-- Enable full-text search extension
-- FTS5 tables will be created separately after main tables

-- ===============================
-- CORE ENTITIES
-- ===============================

-- Users table
CREATE TABLE users (
    id TEXT PRIMARY KEY NOT NULL,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    settings TEXT DEFAULT '{}', -- JSON field
    
    -- Constraints
    CHECK (length(username) >= 3 AND length(username) <= 50),
    CHECK (length(email) >= 5 AND length(email) <= 255),
    CHECK (length(display_name) >= 1 AND length(display_name) <= 100)
);

-- Task lists table
CREATE TABLE task_lists (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    owner_id TEXT NOT NULL,
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    color TEXT DEFAULT '#007AFF',
    icon TEXT DEFAULT 'list',
    parent_id TEXT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    settings TEXT DEFAULT '{}', -- JSON field
    
    -- Foreign keys
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES task_lists(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (length(name) >= 1 AND length(name) <= 200),
    CHECK (position >= 0)
);

-- Locations table
CREATE TABLE locations (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    address TEXT DEFAULT '',
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    radius INTEGER NOT NULL DEFAULT 100,
    category TEXT DEFAULT 'other',
    place_id TEXT NULL,
    metadata TEXT DEFAULT '{}', -- JSON field
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign keys
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (length(name) >= 1 AND length(name) <= 200),
    CHECK (latitude >= -90 AND latitude <= 90),
    CHECK (longitude >= -180 AND longitude <= 180),
    CHECK (radius > 0)
);

-- Tasks table
CREATE TABLE tasks (
    id TEXT PRIMARY KEY NOT NULL,
    title TEXT NOT NULL,
    description TEXT DEFAULT '',
    creator_id TEXT NOT NULL,
    assignee_id TEXT NULL,
    list_id TEXT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    priority INTEGER NOT NULL DEFAULT 3,
    estimated_minutes INTEGER NULL,
    due_at DATETIME NULL,
    completed_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT DEFAULT '{}', -- JSON field
    recurrence_rule TEXT NULL, -- RFC 5545 RRULE
    parent_task_id TEXT NULL,
    
    -- Foreign keys
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (assignee_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (list_id) REFERENCES task_lists(id) ON DELETE SET NULL,
    FOREIGN KEY (parent_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (length(title) >= 1 AND length(title) <= 500),
    CHECK (priority >= 1 AND priority <= 5),
    CHECK (estimated_minutes IS NULL OR estimated_minutes > 0),
    CHECK (status IN ('pending', 'active', 'completed', 'cancelled', 'blocked'))
);

-- ===============================
-- RELATIONSHIP TABLES
-- ===============================

-- Task-Location many-to-many relationship
CREATE TABLE task_locations (
    id TEXT PRIMARY KEY NOT NULL,
    task_id TEXT NOT NULL,
    location_id TEXT NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign keys
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE CASCADE,
    
    -- Unique constraint
    UNIQUE(task_id, location_id)
);

-- Task dependencies
CREATE TABLE task_dependencies (
    id TEXT PRIMARY KEY NOT NULL,
    task_id TEXT NOT NULL,
    depends_on_task_id TEXT NOT NULL,
    dependency_type TEXT NOT NULL DEFAULT 'blocking',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign keys
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (depends_on_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (dependency_type IN ('blocking', 'related', 'scheduled')),
    CHECK (task_id != depends_on_task_id), -- Prevent self-dependency
    
    -- Unique constraint
    UNIQUE(task_id, depends_on_task_id)
);

-- Calendar events
CREATE TABLE calendar_events (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    external_id TEXT NOT NULL,
    title TEXT NOT NULL,
    start_at DATETIME NOT NULL,
    end_at DATETIME NOT NULL,
    location TEXT NULL,
    is_all_day BOOLEAN NOT NULL DEFAULT FALSE,
    is_busy BOOLEAN NOT NULL DEFAULT TRUE,
    metadata TEXT DEFAULT '{}', -- JSON field
    last_synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign keys
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (start_at < end_at),
    UNIQUE(user_id, provider_id, external_id)
);

-- User contexts (snapshots for filtering decisions)
CREATE TABLE contexts (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    current_latitude REAL NULL,
    current_longitude REAL NULL,
    current_location_id TEXT NULL,
    available_minutes INTEGER NOT NULL DEFAULT 0,
    social_context TEXT DEFAULT 'alone',
    energy_level INTEGER NOT NULL DEFAULT 3,
    weather_condition TEXT NULL,
    traffic_level TEXT NULL,
    metadata TEXT DEFAULT '{}', -- JSON field
    
    -- Foreign keys
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (current_location_id) REFERENCES locations(id) ON DELETE SET NULL,
    
    -- Constraints
    CHECK (current_latitude IS NULL OR (current_latitude >= -90 AND current_latitude <= 90)),
    CHECK (current_longitude IS NULL OR (current_longitude >= -180 AND current_longitude <= 180)),
    CHECK (available_minutes >= 0),
    CHECK (energy_level >= 1 AND energy_level <= 5)
);

-- List members (for shared lists)
CREATE TABLE list_members (
    id TEXT PRIMARY KEY NOT NULL,
    list_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer',
    invited_by TEXT NOT NULL,
    invited_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    accepted_at DATETIME NULL,
    
    -- Foreign keys
    FOREIGN KEY (list_id) REFERENCES task_lists(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (role IN ('owner', 'editor', 'viewer')),
    
    -- Unique constraint
    UNIQUE(list_id, user_id)
);

-- Task assignments (delegation tracking)
CREATE TABLE task_assignments (
    id TEXT PRIMARY KEY NOT NULL,
    task_id TEXT NOT NULL,
    assigned_by TEXT NOT NULL,
    assigned_to TEXT NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'pending',
    response_at DATETIME NULL,
    response_message TEXT NULL,
    
    -- Foreign keys
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_to) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (status IN ('pending', 'accepted', 'rejected')),
    CHECK (assigned_by != assigned_to) -- Can't assign to yourself
);

-- ===============================
-- AUDIT AND ANALYTICS TABLES
-- ===============================

-- Filter audit (transparency tracking)
CREATE TABLE filter_audit (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    context_id TEXT NOT NULL,
    is_visible BOOLEAN NOT NULL,
    reasons TEXT NOT NULL, -- JSON array of rule applications
    priority_score REAL NOT NULL DEFAULT 0.0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign keys
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (context_id) REFERENCES contexts(id) ON DELETE CASCADE
);

-- Analytics (daily aggregates)
CREATE TABLE analytics (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL,
    date DATE NOT NULL,
    tasks_created INTEGER NOT NULL DEFAULT 0,
    tasks_completed INTEGER NOT NULL DEFAULT 0,
    tasks_cancelled INTEGER NOT NULL DEFAULT 0,
    minutes_estimated INTEGER NOT NULL DEFAULT 0,
    minutes_actual INTEGER NOT NULL DEFAULT 0,
    location_changes INTEGER NOT NULL DEFAULT 0,
    metadata TEXT DEFAULT '{}', -- JSON field for additional metrics
    
    -- Foreign keys
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (tasks_created >= 0),
    CHECK (tasks_completed >= 0),
    CHECK (tasks_cancelled >= 0),
    CHECK (minutes_estimated >= 0),
    CHECK (minutes_actual >= 0),
    CHECK (location_changes >= 0),
    
    -- Unique constraint
    UNIQUE(user_id, date)
);

-- ===============================
-- INDEXES FOR PERFORMANCE
-- ===============================

-- User indexes
CREATE UNIQUE INDEX idx_users_username ON users(username);
CREATE UNIQUE INDEX idx_users_email ON users(email);

-- Task indexes
CREATE INDEX idx_tasks_assignee_status ON tasks(assignee_id, status);
CREATE INDEX idx_tasks_list_status ON task_lists(id, status);
CREATE INDEX idx_tasks_due_at ON tasks(due_at) WHERE due_at IS NOT NULL;
CREATE INDEX idx_tasks_creator ON tasks(creator_id);
CREATE INDEX idx_tasks_status ON tasks(status);

-- Task relationship indexes
CREATE INDEX idx_task_locations_task ON task_locations(task_id);
CREATE INDEX idx_task_locations_location ON task_locations(location_id);
CREATE INDEX idx_task_dependencies_task ON task_dependencies(task_id);
CREATE INDEX idx_task_dependencies_depends ON task_dependencies(depends_on_task_id);

-- Location indexes
CREATE INDEX idx_locations_user ON locations(user_id);
CREATE INDEX idx_locations_category ON locations(category);

-- Calendar indexes
CREATE INDEX idx_calendar_events_user_time ON calendar_events(user_id, start_at, end_at);
CREATE INDEX idx_calendar_events_provider ON calendar_events(provider_id, external_id);

-- Context indexes
CREATE INDEX idx_contexts_user_timestamp ON contexts(user_id, timestamp);
CREATE INDEX idx_contexts_location ON contexts(current_location_id);

-- List indexes
CREATE INDEX idx_task_lists_owner ON task_lists(owner_id);
CREATE INDEX idx_list_members_list ON list_members(list_id);
CREATE INDEX idx_list_members_user ON list_members(user_id);

-- Audit indexes
CREATE INDEX idx_filter_audit_user_task ON filter_audit(user_id, task_id, created_at);
CREATE INDEX idx_filter_audit_context ON filter_audit(context_id);

-- Analytics indexes
CREATE INDEX idx_analytics_user_date ON analytics(user_id, date);
CREATE INDEX idx_analytics_date ON analytics(date);

-- ===============================
-- FULL-TEXT SEARCH TABLES
-- ===============================

-- Full-text search for tasks
CREATE VIRTUAL TABLE tasks_fts USING fts5(
    title, 
    description, 
    content='tasks', 
    content_rowid='rowid',
    tokenize='porter'
);

-- Triggers to keep FTS in sync with tasks table
CREATE TRIGGER tasks_fts_insert AFTER INSERT ON tasks BEGIN
    INSERT INTO tasks_fts(rowid, title, description) 
    VALUES (new.rowid, new.title, new.description);
END;

CREATE TRIGGER tasks_fts_delete AFTER DELETE ON tasks BEGIN
    DELETE FROM tasks_fts WHERE rowid = old.rowid;
END;

CREATE TRIGGER tasks_fts_update AFTER UPDATE ON tasks BEGIN
    DELETE FROM tasks_fts WHERE rowid = old.rowid;
    INSERT INTO tasks_fts(rowid, title, description) 
    VALUES (new.rowid, new.title, new.description);
END;

-- Full-text search for locations
CREATE VIRTUAL TABLE locations_fts USING fts5(
    name, 
    address, 
    content='locations', 
    content_rowid='rowid',
    tokenize='porter'
);

-- Triggers to keep FTS in sync with locations table
CREATE TRIGGER locations_fts_insert AFTER INSERT ON locations BEGIN
    INSERT INTO locations_fts(rowid, name, address) 
    VALUES (new.rowid, new.name, new.address);
END;

CREATE TRIGGER locations_fts_delete AFTER DELETE ON locations BEGIN
    DELETE FROM locations_fts WHERE rowid = old.rowid;
END;

CREATE TRIGGER locations_fts_update AFTER UPDATE ON locations BEGIN
    DELETE FROM locations_fts WHERE rowid = old.rowid;
    INSERT INTO locations_fts(rowid, name, address) 
    VALUES (new.rowid, new.name, new.address);
END;

-- ===============================
-- TRIGGERS FOR UPDATED_AT
-- ===============================

-- Update triggers for timestamps
CREATE TRIGGER users_updated_at AFTER UPDATE ON users BEGIN
    UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = new.id;
END;

CREATE TRIGGER tasks_updated_at AFTER UPDATE ON tasks BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = new.id;
END;

CREATE TRIGGER task_lists_updated_at AFTER UPDATE ON task_lists BEGIN
    UPDATE task_lists SET updated_at = CURRENT_TIMESTAMP WHERE id = new.id;
END;

CREATE TRIGGER locations_updated_at AFTER UPDATE ON locations BEGIN
    UPDATE locations SET updated_at = CURRENT_TIMESTAMP WHERE id = new.id;
END;