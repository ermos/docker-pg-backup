# PostgreSQL Backup

Automated PostgreSQL backup service with scheduled backups and multiple storage backends support.

https://github.com/ermos/docker-pg-backup

## Quick Start

```bash
docker run -d \
  -e PGHOST=your-postgres-host \
  -e PGPASSWORD=your-password \
  -e BACKUP_CRON="0 2 * * *" \
  -v /path/to/backups:/backups \
  ermos/pg-backup:17
```

## Available Tags

- `ermos/pg-backup:17` - PostgreSQL 17 client
- `ermos/pg-backup:16` - PostgreSQL 16 client
- `ermos/pg-backup:15` - PostgreSQL 15 client

Use the tag matching your PostgreSQL server version.

## Environment Variables

### PostgreSQL Connection

| Variable | Description | Default |
|----------|-------------|---------|
| `PGHOST` | PostgreSQL host | `localhost` |
| `PGPORT` | PostgreSQL port | `5432` |
| `PGUSER` | PostgreSQL user | `postgres` |
| `PGPASSWORD` | PostgreSQL password | *required* |
| `PGDATABASE` | Database to backup | `postgres` |

### Backup Options

| Variable | Description | Default |
|----------|-------------|---------|
| `BACKUP_CRON` | Cron expression for scheduled backups | *required* |
| `BACKUP_ON_START` | Run backup immediately on container start | `false` |
| `BACKUP_TIMEOUT` | Backup timeout in minutes | `30` |
| `BACKUP_COMPRESSION` | Enable gzip compression (plain format only) | `true` |
| `PGDUMP_FORMAT` | pg_dump format: `plain`, `custom`, `directory`, `tar` | `custom` |
| `PGDUMP_OPTIONS` | Additional pg_dump options | |

### Storage Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `STORAGE_TYPE` | Storage backend: `local`, `s3`, `gcp` | `local` |
| `RETENTION_COUNT` | Number of backups to keep (0 = unlimited) | `0` |

### Local Storage

| Variable | Description | Default |
|----------|-------------|---------|
| `LOCAL_BACKUP_PATH` | Path to store backups | `/backups` |

### S3 Storage (AWS S3, MinIO, etc.)

| Variable | Description | Default |
|----------|-------------|---------|
| `S3_ENDPOINT` | S3 endpoint URL (for MinIO/custom S3) | |
| `S3_REGION` | AWS region | `us-east-1` |
| `S3_BUCKET` | S3 bucket name | *required for S3* |
| `S3_ACCESS_KEY` | S3 access key | *required for S3* |
| `S3_SECRET_KEY` | S3 secret key | *required for S3* |
| `S3_PATH_STYLE` | Use path-style URLs (for MinIO) | `false` |
| `S3_BACKUP_PREFIX` | Prefix for backup files in bucket | |

### Google Cloud Storage

| Variable | Description | Default |
|----------|-------------|---------|
| `GCS_BUCKET` | GCS bucket (format: `gs://bucket-name/prefix`) | *required for GCS* |
| `GCP_CREDENTIALS_FILE` | Path to service account JSON file | *required for GCS* |

## Examples

### Daily backup to local volume

```bash
docker run -d \
  -e PGHOST=postgres \
  -e PGPASSWORD=secret \
  -e BACKUP_CRON="0 2 * * *" \
  -v /backups:/backups \
  ermos/pg-backup:17
```

### Backup to AWS S3

```bash
docker run -d \
  -e PGHOST=postgres \
  -e PGPASSWORD=secret \
  -e BACKUP_CRON="0 */6 * * *" \
  -e STORAGE_TYPE=s3 \
  -e S3_BUCKET=my-backups \
  -e S3_REGION=eu-west-1 \
  -e S3_ACCESS_KEY=AKIA... \
  -e S3_SECRET_KEY=... \
  -e S3_BACKUP_PREFIX=postgres \
  -e RETENTION_COUNT=30 \
  ermos/pg-backup:17
```

### Backup to MinIO

```bash
docker run -d \
  -e PGHOST=postgres \
  -e PGPASSWORD=secret \
  -e BACKUP_CRON="0 0 * * *" \
  -e STORAGE_TYPE=s3 \
  -e S3_ENDPOINT=http://minio:9000 \
  -e S3_BUCKET=backups \
  -e S3_ACCESS_KEY=minioadmin \
  -e S3_SECRET_KEY=minioadmin \
  -e S3_PATH_STYLE=true \
  ermos/pg-backup:17
```

### Backup to Google Cloud Storage

```bash
docker run -d \
  -e PGHOST=postgres \
  -e PGPASSWORD=secret \
  -e BACKUP_CRON="0 3 * * *" \
  -e STORAGE_TYPE=gcs \
  -e GCS_BUCKET=gs://my-bucket/backups \
  -v /path/to/credentials.json:/credentials.json:ro \
  -e GCP_CREDENTIALS_FILE=/credentials.json \
  ermos/pg-backup:17
```

### Docker Compose

```yaml
services:
  postgres:
    image: postgres:17
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data

  backup:
    image: ermos/pg-backup:17
    environment:
      PGHOST: postgres
      PGPASSWORD: secret
      BACKUP_CRON: "0 2 * * *"
      BACKUP_ON_START: "true"
      RETENTION_COUNT: 7
    volumes:
      - ./backups:/backups
    depends_on:
      - postgres

volumes:
  pgdata:
```

## Cron Expression Format

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sunday=0)
│ │ │ │ │
* * * * *
```

Examples:
- `0 2 * * *` - Daily at 2:00 AM
- `0 */6 * * *` - Every 6 hours
- `0 0 * * 0` - Weekly on Sunday at midnight
- `0 0 1 * *` - Monthly on the 1st at midnight

## pg_dump Formats

| Format | Description |
|--------|-------------|
| `custom` | Compressed, flexible restore options (recommended) |
| `plain` | Plain SQL script, can be compressed with `BACKUP_COMPRESSION` |
| `tar` | tar archive, suitable for pg_restore |
| `directory` | Directory with one file per table |

## License

MIT
