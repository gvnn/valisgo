# valisgo

🧳 A suitcase full of wonders

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

3. **Start the development server** (with hot-reload):
   ```bash
   make dev
   ```

### Database Migrations (Atlas)

Migrations are generated and applied via Atlas.

- **Generate new migrations** (after changing schema):
  ```bash
  MIGRATION_NAME="add_users_table" make migrate-diff
  ```
- **Apply migrations**:
  ```bash
  make migrate-apply        # Applies to Postgres
  ```
