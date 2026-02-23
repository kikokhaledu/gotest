# React Frontend

UI application for the test system.

Data flow:

`React (5173) -> Node API gateway (3000) -> Go backend (8080)`

## Setup

Install dependencies:

```bash
npm install
```

Run development server:

```bash
npm run dev
```

Frontend runs on `http://localhost:5173` by default.

## Configuration

The frontend can use these Vite env variables:

- `VITE_API_URL` (default: `http://localhost:3000`)
- `VITE_AUTH_USERNAME` (default: `admin`)
- `VITE_AUTH_PASSWORD` (default: `ChangeMe123@`)
- `VITE_API_KEY` (optional pre-seeded API key)
- `VITE_ACTOR_NAME` (default: `admin`)

For Docker Compose workflow, these values are managed from the root `.env`.
If you run the frontend standalone, you can create `react-frontend/.env` with the same keys.

Important: `VITE_*` values are embedded in the client bundle at build time. Do not place real secrets in them.

## Features

- Health status display
- Users list with user selection
- Tasks list with filtering by status
- Create user form
- Create task form
- Update task form
- Task history modal (shows full audit history)
- Stats dashboard

## Testing

```bash
npm test
```

## Production Build

```bash
npm run build
```

Built output is written to `dist/`.
