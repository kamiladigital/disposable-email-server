# Email Server

[![CI](https://github.com/habibiefaried/email-server/actions/workflows/ci.yml/badge.svg)](https://github.com/habibiefaried/email-server/actions/workflows/ci.yml)

A simple SMTP server in Go using the `go-smtp` library.

## Features
- Accepts SMTP connections on port 25
- Logs sender, recipient, and email body to console
- Stores emails to PostgreSQL database (raw content + HTML body)
- UUIDv7 primary keys (timestamp-sortable, no guessable IDs)
- Base64 email content decoded before database insertion
- HTML body is generated from raw MIME content using enmime (inline images embedded as data URIs)
- Fallback to file storage when database is unavailable
- HTTP API: `/inbox` (summary list) and `/email` (full detail)
- Fully tested with automated CI/CD pipeline
- Does NOT forward emails (for testing/learning only)

## Quick Start

### 1. Build
```sh
make build
```

### 2. Run (No root required by default)
```sh
# Linux/Mac - uses default port 2525 (no sudo needed)
./email-server

# Or use port 25 (requires sudo)
sudo SMTP_PORT=25 ./email-server

# Windows (no administrator required by default)
$env:SMTP_PORT=2525; .\email-server.exe
```


## Environment Variables
| Variable      | Required | Description                                                      |
|---------------|----------|------------------------------------------------------------------|
| SMTP_PORT     | No       | (Optional) SMTP server port. Defaults to `2525` if not set. Use port `25` for production or when running as root.       |
| HTTP_PORT     | No       | (Optional) HTTP health port. Defaults to `48080` if not set.      |
| EMAIL_SIZE_LIMIT | No    | (Optional) Maximum email size in bytes before rejection. Defaults to `524288` (512KB). Emails exceeding this limit will be stored with error message: "Sorry, the email exceeds our limit (512kb)". This is a soft limit checked before expensive MIME parsing to prevent memory exhaustion. Set to `0` to disable limit.       |
| DB_URL        | No       | (Optional) PostgreSQL connection string (works with Neon, AWS RDS, or any PostgreSQL). If provided, emails are saved to database only. Falls back to file storage if connection fails. Format: `user=username password=pass dbname=emaildb host=hostname port=5432 sslmode=require`       |

**Note:** For Neon PostgreSQL, always use `sslmode=require`. For local PostgreSQL, you can use `sslmode=disable`.

## Project Structure
- `cmd/email-server/main.go` — Entry point
- `internal/server/` — SMTP backend/session/server logic
- `internal/dnsutil/` — DNS validation and checking

## Clean Up
To remove the binary:
```sh
make clean
```

## Live Testing & Screenshots

## HTTP API Endpoints

The server exposes HTTP endpoints on `HTTP_PORT` (default `48080`):

### Health Check
- **Endpoint:** `GET /`
- **Description:** Returns server health status
- **Response:** Plain text `OK`
- **Example:**
  ```bash
  curl http://localhost:48080/
  ```

### Inbox API (Summary List)
- **Endpoint:** `GET /inbox?email=<address>&page=<n>`
- **Description:** Fetch email summaries (no body or attachments) for a recipient, sorted by received timestamp descending
- **Query Parameters:**
  - `email` (required) — Recipient email address to filter by
  - `page` (optional) — Page number, 1-based (default: 1). Each page returns 5 emails.
- **Response:** JSON array of up to **5** email summaries per page
- **CORS:** Enabled for cross-origin requests (React/frontend integration)
- **Requires:** PostgreSQL storage must be configured (`DB_URL` environment variable)
- **Examples:**
  ```bash
  # Get latest 5 emails (page 1)
  curl http://localhost:48080/inbox?email=test@example.com
  
  # Get emails 6-10 (page 2)
  curl http://localhost:48080/inbox?email=test@example.com&page=2
  ```
- **Response Format:**
  ```json
  [
    {
      "id": "0194d3f0-7e1a-7b12-9a3f-4c5d6e7f8a9b",
      "from": "sender@example.com",
      "to": "test@example.com",
      "subject": "Test Email",
      "date": "Wed, 5 Feb 2026 10:30:00 +0000",
      "created_at": "2026-02-06T08:30:00Z"
    }
  ]
  ```

### Email Detail API
- **Endpoint:** `GET /email?id=<uuidv7>`
- **Description:** Fetch full email detail including HTML-rendered body by UUIDv7 ID. The server parses raw MIME content using enmime and embeds inline images as data URIs.
- **Query Parameters:**
  - `id` (required) — UUIDv7 of the email
- **Response:** JSON object with full email content
- **CORS:** Enabled for cross-origin requests
- **Examples:**
  ```bash
  # Get full email detail
  curl http://localhost:48080/email?id=0194d3f0-7e1a-7b12-9a3f-4c5d6e7f8a9b
  ```
- **Response Format:**
  ```json
  {
    "id": "0194d3f0-7e1a-7b12-9a3f-4c5d6e7f8a9b",
    "from": "sender@example.com",
    "to": "test@example.com",
    "subject": "Test Email",
    "date": "Wed, 5 Feb 2026 10:30:00 +0000",
    "body": "<!DOCTYPE html><html>...rendered HTML with inline images...</html>",
    "created_at": "2026-02-06T08:30:00Z"
  }
  ```

**Note:** The `/inbox` endpoint returns **5 emails per page** (no body) for fast listing. Use `/email?id=<uuid>` to fetch the full HTML body of a specific email. The server uses enmime to parse raw MIME content and embeds inline images as data URIs. All IDs use UUIDv7 format (timestamp-sortable, non-guessable). Base64-encoded email content is automatically decoded before storage.

### Domain Validation API
- **Endpoint:** `GET /domain/validate?email=<address>`
- **Description:** Validate that a domain has correct DNS records for email delivery. Checks if the domain's MX records point to mail servers that resolve to `149.28.152.71`. This supports both direct MX records (domain → IP) and standard mail server setups (domain → mx1.domain → IP). Performs live lookups without caching.
- **Query Parameters:**
  - `email` (required) — Email address to validate (e.g., `user@example.com`)
- **Response:** JSON object with validation status
- **CORS:** Enabled for cross-origin requests
- **Examples:**
  ```bash
  # Validate a domain
  curl http://localhost:48080/domain/validate?email=user@example.com
  ```
- **Success Response Format:**
  ```json
  {
    "status": "ok",
    "domain": "example.com",
    "mx_status": "✓ OK (MX: mx1.example.com → 149.28.152.71)"
  }
  ```
- **Failure Response Format:**
  ```json
  {
    "status": "error",
    "domain": "example.com",
    "message": "MX records do not resolve to expected IP (149.28.152.71)",
    "mx_status": "✗ FAILED (MX records mail.example.com do not resolve to 149.28.152.71)"
  }
  ```

**Note:** This endpoint performs DNS lookups directly without caching. The validation checks if any MX record for the domain resolves to `149.28.152.71`. This supports standard email configurations where MX records point to dedicated mail server subdomains (e.g., `mx1.domain.com`, `mail.domain.com`).


Below are real-world screenshots and explanations of the server in action:

### 1. Setup & Receiving Email

![Server Startup and DNS Verification](screenshots/1.png)

- The server is started with the default SMTP port (2525) for local testing.
- Emails are accepted and persisted using either PostgreSQL or file storage.

### 2.1. Sending an Email from Gmail

![Sending from Gmail](screenshots/2.2.png)

- An email is sent from Gmail to the server's configured address.

### 2.2. Email Received and Saved

![Email Saved to File](screenshots/2.1.png)
![Email content](screenshots/2.3.png)

- The server receives the email and saves it to a file in the `emails/` directory.
- The filename is a timestamp in nanoseconds, ensuring uniqueness.
- The server logs a summary: `from: <sender>, saved in <filename>`
- No email content is printed to the console for privacy and clarity.

## Docker Deployment

### Prerequisites

Create a `.env` file for your environment variables (keeps sensitive data out of git):

```bash
# Copy the example file
cp .env.example .env

# Edit with your configuration
nano .env  # or use your preferred editor
```

Example `.env` file for Neon PostgreSQL:
```bash
# SMTP Configuration
SMTP_PORT=2525
HTTP_PORT=48080

# Neon PostgreSQL Connection
DB_URL=user=myuser password=mypassword dbname=mydb host=ep-something.us-east-1.aws.neon.tech port=5432 sslmode=require
```

**Important:** Add `.env` to your `.gitignore` to avoid committing credentials.

### Build and Run with Docker Compose

```bash
# Clone or download the repository
cd email-server

# Create your .env file (see above)
cp .env.example .env
# Edit .env with your Neon credentials

# Build and start the service
docker-compose up -d

# View logs
docker-compose logs -f email-server

# Stop the service
docker-compose down
```

### Docker Build Notes (gcc/cgo)

The Docker build is configured with `CGO_ENABLED=0`, so it does not require a C compiler. If you previously saw `C compiler "gcc" not found`, rebuild with the latest Dockerfile:

```bash
docker-compose build --no-cache
docker-compose up -d
```

### Manual Docker Build

```bash
# Build the image
docker build -t email-server .

# Run with environment variables (option 1: inline)
docker run -p 25:2525 -p 48080:48080 \
  -e SMTP_PORT=2525 \
  -e DB_URL="user=myuser password=mypass dbname=mydb host=ep-something.us-east-1.aws.neon.tech port=5432 sslmode=require" \
  email-server

# Run with .env file (option 2: recommended for security)
docker run -p 25:2525 -p 48080:48080 \
  --env-file .env \
  email-server
```

## Database Schema

When `DB_URL` is provided, the server automatically creates two tables:

### email table
Stores email metadata and content:
- `id` (UUID PRIMARY KEY) — UUIDv7 via `github.com/google/uuid` (timestamp-sortable, non-guessable)
- `from` (TEXT) — Sender email address
- `to` (TEXT) — Recipient email address  
- `subject` (TEXT) — Email subject
- `date` (TEXT) — Email send date
- `body` (TEXT) — Parsed email body (base64 decoded)
- `html_body` (TEXT) — HTML body (base64 decoded)
- `raw_content` (TEXT) — Full raw email content
- `created_at` (TIMESTAMP) — Record creation time

### attachment table
Stores email attachments:
- `id` (UUID PRIMARY KEY) — UUIDv7
- `email_id` (UUID FK) — Foreign key to email table
- `filename` (TEXT) — Original filename
- `content_type` (TEXT) — MIME type (e.g., image/png)
- `data` (BYTEA) — Binary attachment data (base64 decoded)
  - **Size limit:** Attachments larger than **2MB** are automatically replaced with a redacted placeholder message to prevent database bloat
- `created_at` (TIMESTAMP) — Record creation time

## Storage Options

### PostgreSQL Storage (Primary)
When `DB_URL` is set, emails are saved exclusively to PostgreSQL database. This is the recommended mode for production use.

### File Storage (Fallback)
Emails are saved to `emails/<to>/<from>/timestamp.txt` only when:
- `DB_URL` is not provided, or
- PostgreSQL connection fails (automatic fallback with warning)

## CI/CD Pipeline

The project includes a comprehensive GitHub Actions workflow that automatically runs on every push and pull request. The CI pipeline:
#### Key Features:
- **Default SMTP Port:** 2525 (no root required, development-friendly)
- **Port Mapping:** In Docker, maps `25:2525` (external:internal)
- **Custom Ports:** Use `SMTP_PORT` environment variable to override default
- **Production Ready:** Set `SMTP_PORT=25` with root privileges for standard SMTP port

#### How it works:
```bash
# Use default port 25 (requires sudo/root)
./email-server

# Use custom port for non-privileged environments (CI, Docker, dev)
SMTP_PORT=2525 ./email-server

# Docker automatically uses port 25 (runs as root inside container)
docker run -e SMTP_PORT=25 email-server
```
### Automated Tests
1. **Unit Tests** - Runs all Go unit tests with race detection and coverage reporting
2. **Integration Tests** - Full end-to-end testing with PostgreSQL
3. **API Testing** - Comprehensive curl-based tests including:

#### Test Coverage:
- ✅ Health check endpoint validation
- ✅ Valid email addresses with data
- ✅ Non-existent addresses (empty results)
- ✅ Missing required parameters (400 error)
- ✅ Pagination with page parameter
- ✅ Negative page handling
- ✅ Empty email addresses
- ✅ CORS headers verification
- ✅ Email detail by UUIDv7 (`/email?id=`)
- ✅ Non-existent email ID (404 error)
- ✅ Missing email ID parameter (400 error)
- ✅ UUIDv7 format validation
- ✅ Custom generated emails (5 unique scenarios)
- ✅ Special characters and Unicode handling
- ✅ Long content storage and retrieval
- ✅ Multi-recipient email isolation
- ✅ Attachment metadata validation

### Test Data
The CI pipeline automatically:
1. Starts PostgreSQL service
2. Loads sample emails via SMTP (gmail.txt, anonymousemail.txt, attachments.txt)
3. Generates 5 custom test emails with dynamic content:
   - Simple text email
   - HTML content email
   - Special characters (Unicode, emojis)
   - Different recipients for isolation testing
   - Long content (50+ lines) for stress testing
4. Verifies data integrity and API responses
5. Tests edge cases and error scenarios

### Running Tests Locally
```bash
# Run unit tests
go test ./... -v -race

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Generate and send test emails (requires swaks)
# Install swaks: brew install swaks (macOS) or apt-get install swaks (Linux)
./scripts/test-emails.sh localhost 25

# Then verify via API
curl "http://localhost:48080/inbox?email=testuser@example.com" | jq

# Get full detail for a specific email
curl "http://localhost:48080/email?id=<uuid-from-inbox>" | jq
```

### Continuous Integration
Every commit is automatically tested against:
- Go 1.25.7
- PostgreSQL 16 Alpine
- Ubuntu Latest runner
- Multiple edge cases and error scenarios

The pipeline ensures code quality and prevents regressions before merging.