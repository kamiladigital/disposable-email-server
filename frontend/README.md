# Email Server Frontend (Read-only)

A React + Vite frontend that provides a modern, read-only console for viewing emails received by the mail server. The UI emphasizes clarity and a left-hand navigation menu.

## Features

- Left sidebar navigation with **Home** and **Inbox**
- Read-only inbox view with message detail panel
- Home overview explaining the system purpose and instructions
- Inbox gate: enter a mailbox address to load messages
- Pagination (Prev / Next) — 5 emails per page via `page` parameter
- Email detail fetched on click via `/email?id=<uuid>`
- Refresh button with loading animation

## API endpoints

| Endpoint | Params | Description |
|---|---|---|
| `GET /inbox` | `email` (required), `page` (page number, default 1) | List emails for a mailbox, 5 per page |
| `GET /email` | `id` (UUID) | Fetch full email content by ID |

Example:
- `http://149.28.152.71:48080/inbox?email=admin@test.milahabibie.com&page=1`
- `http://149.28.152.71:48080/email?id=019c343a-5282-773a-93a3-49837ab3c4c5`

## Project structure (high-level)

- `index.html`: app shell and font setup
- `src/main.jsx`: React entry point
- `src/App.jsx`: layout, API integration, and UI logic
- `src/index.css`: styling and layout theme

## Run in development mode

1. Install dependencies:
   - `npm install`
2. Start the dev server:
   - `npm run dev`

Vite will print the local URL in the terminal. Open it in your browser.

## Build static files (HTML/CSS/JS)

1. Install dependencies:
   - npm install
2. Build the production assets:
   - npm run build

The static site will be generated in the dist folder.

## Host on S3 (static website)

1. Upload the contents of the dist folder to your S3 bucket.
2. Enable S3 static website hosting for the bucket.
3. Set the index document to index.html (and error document to index.html if you want SPA routing).
