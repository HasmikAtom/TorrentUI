# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TorrentUI is a self-hosted web app for managing torrent downloads with Plex integration. It has a React frontend and Go backend that communicates with a Transmission BitTorrent daemon. Users can paste magnet links, upload .torrent files, or search ThePirateBay/RuTracker directly from the UI.

## Hosting & Authentication

The app is hosted on a personal server and exposed to the internet via a Cloudflare Tunnel. Authentication is handled with a simple one-time PIN mechanism.

## Architecture

```
Frontend (React+Vite, :5173 dev / :3000 prod)
  → /api proxy →
Backend (Go+Gin, :8085 dev / :8080 prod)
  → Transmission RPC (:9091)
  → Chromedp headless browser (scraping TPB/RuTracker)
  → /mediastorage/{Movies,Series,Music} → Plex
```

- **Frontend** proxies all `/api` requests to the backend via Vite's proxy config, stripping the `/api` prefix
- **Backend** uses Chromedp browser pool (singleton via `sync.Once`) for scraping, with SSE streaming for real-time search results
- **Prepare-Edit-Finalize flow**: Torrents go through prepare → poll metadata → user edits name → finalize before downloading

## Development Commands

### Frontend (`cd frontend`)
```bash
npm run dev        # Vite dev server on :5173 with HMR
npm run build      # TypeScript check + Vite production build
npm run lint       # ESLint
npm run preview    # Serve production build on :3000
```

### Backend (`cd backend`)
```bash
go run .           # Run directly
air                # Hot reload (configured via .air.toml)
```

### Docker (from project root)
```bash
make dev-build     # Build and start dev containers (hot reload)
make dev-logs      # Stream dev logs
make dev-down      # Stop dev containers
make prod-build-deploy  # Build and deploy production
make clean-dev     # Remove dev containers, images, volumes
make clean-prod    # Remove prod containers, images, volumes
```

## Project Structure

```
backend/
  main.go              # Entry point: routes, Transmission client init, graceful shutdown
  config/              # Environment-based config (DEV_* / PROD_* prefixed vars)
  transmission/        # Transmission RPC client implementation
  scraper/             # Chromedp browser pool, piratebay.go, rutracker.go, auth.go

frontend/
  src/
    App.tsx            # Main layout with tabbed interface (Download, PirateBay, RuTracker)
    components/
      TorrentDownloader.tsx  # Magnet/file upload with prepare-edit-finalize flow
      TorrentList.tsx        # Active torrents with 3s polling, rename/delete
      ScraperUI.tsx          # Search interface with SSE streaming progress
      ScrapedTorrents.tsx    # Search results with batch selection
      StorageInfo.tsx        # Storage usage visualization (30s polling)
      ui/                    # Radix-based shadcn-style primitives
```

## Key Technical Details

- **Path alias**: `@/` maps to `frontend/src/` (configured in vite.config.ts and tsconfig)
- **Styling**: Tailwind CSS with CSS variables for theming (dark/light mode), custom slide animations
- **UI library**: shadcn-style components built on Radix UI primitives in `frontend/src/components/ui/`
- **Backend env**: Copy `backend/.env.example` to `backend/.env`; uses `DEV_` prefix for local dev, `PROD_` for Docker
- **API proxy**: Dev server proxies `/api` to `VITE_API_TARGET` (default `http://localhost:8085`); production proxies to `http://backend:8080`
- **SSE endpoints**: `/scrape/piratebay/:name/stream` and `/scrape/rutracker/:name/stream` use Server-Sent Events
- **No test suite**: The project currently has no automated tests
