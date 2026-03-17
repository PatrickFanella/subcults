# Disaster Recovery

## Overview

This document covers backup, restore, and disaster recovery procedures for Subcults. The primary data store is Neon Postgres (managed), which provides native PITR.

## Recovery Targets

| Metric                             | Target   | Notes                    |
| ---------------------------------- | -------- | ------------------------ |
| **RTO** (Recovery Time Objective)  | < 1 hour | Time to restore service  |
| **RPO** (Recovery Point Objective) | < 1 hour | Maximum data loss window |

## Backup Strategy

### Neon Postgres (Primary)

Neon provides built-in backup capabilities:

- **Automatic branching**: Create instant database branches for testing/staging
- **Point-in-Time Recovery (PITR)**: Restore to any point within the retention window
- **Zero-downtime**: Branch operations don't affect the primary database

To create a branch (instant backup):

```bash
# Via Neon CLI
neonctl branches create --project-id <id> --name backup-$(date +%Y%m%d)
```

### Manual Backups (Supplementary)

For additional safety, run manual `pg_dump` backups:

```bash
# Create a compressed backup
./scripts/backup.sh

# Backup to a specific file
./scripts/backup.sh /path/to/backup.sql.gz
```

Backups are stored in `backups/` by default. The script:

- Compresses with gzip
- Verifies integrity after creation
- Retains the last 30 backups (auto-cleanup)

### Backup Schedule

| Type           | Frequency    | Retention            | Location       |
| -------------- | ------------ | -------------------- | -------------- |
| Neon PITR      | Continuous   | Per plan (7-30 days) | Neon cloud     |
| Manual pg_dump | Daily (cron) | 30 days              | VPS `backups/` |

Set up daily cron:

```bash
# Add to crontab -e
0 3 * * * /home/onnwee/projects/subcults/scripts/backup.sh >> /var/log/subcults-backup.log 2>&1
```

## Restore Procedures

### Scenario 1: Restore from Neon PITR

Best for: accidental data deletion, corruption within the retention window.

1. Go to Neon dashboard → Project → Branches
2. Create a new branch from the desired point in time
3. Update `DATABASE_URL` in `deploy/.env` to point to the new branch
4. Restart services:
   ```bash
   cd deploy && docker compose up -d --force-recreate
   ```
5. Verify data integrity
6. Once confirmed, promote the branch or migrate data back

### Scenario 2: Restore from pg_dump backup

Best for: full database rebuild, migration to new provider.

```bash
# List available backups
ls -lh backups/

# Restore (interactive — requires typing 'RESTORE' to confirm)
./scripts/restore.sh backups/subcults_20260221_120000.sql.gz

# Apply any pending migrations
make migrate-up-prod

# Verify
curl http://localhost:8080/health/ready
```

### Scenario 3: Complete infrastructure rebuild

If the VPS is lost:

1. **Provision new VPS** with Docker and Docker Compose
2. **Clone repositories**:
   ```bash
   git clone git@github.com:subculture-collective/subcults.git
   # Also clone caddy and monitoring repos
   ```
3. **Create Docker network**: `docker network create web`
4. **Start Caddy**: `cd ~/projects/caddy && docker compose up -d`
5. **Start monitoring**: `cd ~/projects/monitoring && docker compose up -d`
6. **Configure environment**: Copy `deploy/.env` from secure storage
7. **Start Subcults**: `cd ~/projects/subcults && ./scripts/deploy.sh`
8. **Verify**: Check health endpoints and monitoring dashboards

## Data Inventory

| Data                             | Location              | Backup Method        | Critical? |
| -------------------------------- | --------------------- | -------------------- | --------- |
| Database (scenes, events, users) | Neon Postgres         | PITR + pg_dump       | Yes       |
| Media uploads (images, audio)    | Cloudflare R2         | R2 replication       | Yes       |
| Configuration/secrets            | deploy/.env           | Manual secure copy   | Yes       |
| Monitoring data (metrics)        | Prometheus volume     | Retention only (30d) | No        |
| Logs                             | Loki volume           | Retention only (90d) | No        |
| Traces                           | Jaeger/Badger volumes | Retention only (30d) | No        |

## Testing

### Monthly Restore Test

Run monthly to verify backups are usable:

1. Create a fresh backup: `./scripts/backup.sh`
2. Spin up a test database (Docker):
   ```bash
   docker run -d --name restore-test -e POSTGRES_PASSWORD=test -p 15432:5432 postgis/postgis:16-3.4-alpine
   export DATABASE_URL=postgres://postgres:test@localhost:15432/postgres?sslmode=disable
   ```
3. Run restore: `./scripts/restore.sh backups/latest.sql.gz`
4. Verify table count and sample data
5. Clean up: `docker rm -f restore-test`

### Verify Neon PITR

1. Create a branch from 1 hour ago
2. Connect and verify recent data is present up to that point
3. Delete the test branch

## Incident Response

If data loss is detected:

1. **Stop writes immediately**: Scale API to 0 or set maintenance mode
2. **Assess scope**: What data is missing? When was it last correct?
3. **Choose recovery method**: Neon PITR for precise recovery, pg_dump for full restore
4. **Execute restore** per procedures above
5. **Verify** data integrity and service health
6. **Post-incident**: Document what happened and update procedures
