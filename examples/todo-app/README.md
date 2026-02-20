# TODO App Example

This example demonstrates how to use Maslow to define, build, and verify a web-based TODO application.

## Stack

- **Backend**: Go HTTP server with REST API
- **Frontend**: React SPA
- **Persistence**: SQLite3

## Quick Start

### 1. Scaffold a new project

```bash
# From the maslow repo root, build the maslow binary
go build -o bin/maslow ./cmd/maslow/

# Scaffold a new project in a directory of your choice
./bin/maslow scaffold todo-app --dir /path/to/your/todo-app --description "A simple web-based TODO application"
```

### 2. Copy the example spec

```bash
# Copy the maslow.yaml from this example into your new project
cp examples/todo-app/maslow.yaml /path/to/your/todo-app/maslow.yaml
```

### 3. Build the app

The maslow.yaml spec defines the structure. An agent (or human) builds the app following the spec:

- Go backend in `cmd/server/` with REST API for TODO CRUD operations
- React frontend in `frontend/` with a simple UI
- SQLite for persistence

### 4. Verify with Maslow

```bash
cd /path/to/your/todo-app

# Quick verification during development
maslow verify --profile quick

# Full verification before merging
maslow verify --profile full
```

## What the Spec Defines

| Section | Purpose |
|---------|---------|
| `checks` | Build and test commands for backend and frontend |
| `profiles` | Quick (build + lint) and full (build + test + lint + frontend) |
| `contracts` | API contract tests for TODO CRUD operations |
| `budgets` | Binary size limit for the backend server |
| `policy` | Protected files and denied paths |
