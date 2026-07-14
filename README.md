# valisgo

🧳 A suitcase full of wonders.

The goal of this project is to provide a free and open-source alternative to Nexus and Jfrog.

## Prerequisites

- **Go**: `1.26.x`
- **[Atlas](https://atlasgo.io/)**: For managing database migrations
- **Docker**: For running PostgreSQL locally

## Quick Start

1. **Install dependencies and tooling** (installs `air` for live-reloading):

   ```bash
   make setup
   ```

2. **Apply database migrations**:

   ```bash
   make migrate-apply
   ```

3. **Seed the database** with initial data:

   ```bash
   make seed
   ```

4. **Start the development server** (with hot-reload):
   ```bash
   make dev
   ```

## CLI Utility

`valisgo` includes a command-line interface (`valisgo-cli`) for interacting with the management API.

To run it directly using Go:
```bash
go run ./cmd/cli --help
```

To build a standalone binary:
```bash
make build-cli
./bin/valisgo-cli --help
```

### Database Migrations (Atlas)

Migrations are generated and applied via Atlas.

- **Generate new migrations** (after changing schema):
  ```bash
  MIGRATION_NAME="add_users_table" make migrate-diff
  ```
  ```bash
  make migrate-apply        # Applies to Postgres
  ```

## Authorization (Casbin)

`valisgo` uses [Casbin](https://casbin.org/) (via `gorm-adapter`) to handle Role-Based Access Control (RBAC).
