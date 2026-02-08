# migrate-remove-attachments

One-time destructive migration to drop the attachment table and truncate the email table.

## Usage

Set DB_URL and run:

```sh
go run .\cmd\migrate-remove-attachments --confirm
```

## What it does

- Drops `attachment` table if it exists
- Truncates `email` table

## Notes

- This is destructive and cannot be undone.
- After running, use the SMTP server to re-ingest emails, or run reprocess-emails to backfill bodies from raw content if you already re-inserted raw emails.
