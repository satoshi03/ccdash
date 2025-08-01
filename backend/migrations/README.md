# CCDash Database Migration System

This directory contains database migrations for the CCDash project.

## Migration File Format

Migration files follow the naming convention:
```
{timestamp}_{description}.{up|down}.sql
```

Example:
- `20250801000000_initial_schema.up.sql` - Creates initial database schema
- `20250801000000_initial_schema.down.sql` - Rolls back initial schema

## Using the Migration Tool

### Check Migration Status
```bash
# Using environment variable
CCDASH_DB_PATH=~/.ccdash/ccdash.db ./bin/migrate status

# Using command line flag
./bin/migrate -db ~/.ccdash/ccdash.db status
```

### Run Pending Migrations
```bash
# Run all pending migrations
CCDASH_DB_PATH=~/.ccdash/ccdash.db ./bin/migrate up
```

### Roll Back Last Migration
```bash
# Roll back the most recent migration
CCDASH_DB_PATH=~/.ccdash/ccdash.db ./bin/migrate down
```

## Creating New Migrations

1. Create two SQL files with matching timestamps:
   - `{timestamp}_{description}.up.sql` - Forward migration
   - `{timestamp}_{description}.down.sql` - Rollback migration

2. Timestamp format: `YYYYMMDDHHMMSS` (e.g., `20250801120000`)

3. Ensure down migrations properly reverse all changes made by up migrations

## Migration History

The system tracks all applied migrations in the `migration_history` table, including:
- Version and name
- Execution time
- Success/failure status
- Applied timestamp

## Phase 1 Implementation

This is Phase 1 of the database migration system, which includes:
- ✅ Basic migration engine with file scanner and executor
- ✅ Version tracking and history management
- ✅ CLI tool with `up`, `down`, and `status` commands
- ✅ Transaction-based execution with rollback on error
- ✅ Support for SQL file migrations

Future phases will add:
- Migration creation command (`migrate create`)
- Go-based migrations
- Dry-run mode
- Advanced rollback strategies