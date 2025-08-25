# Wordle – Extended Implementation

A multi-phase implementation of the Wordle game, built as part of a programming assignment.  
The project demonstrates the ability to understand abstract problems, make design trade-offs, and build scalable client/server systems in both **TypeScript/Express** and **Go**.

---

## Overview

This repository implements the programming assignment tasks:

1. **Normal Wordle** – classic single-player game.  
2. **Server/Client Wordle** – separation of concerns between frontend and backend.  
3. **Host Cheating Wordle** – variant based on *Absurdle*, where the host delays answer selection.  
4. **Multi-player Wordle** – executed on a daily challenge mode and extended design and planned features, including **SWORDLE**, a competitive real-time mode.

The solution started as a **TypeScript/Express server** and was later **refactored to Go** for performance, clarity, and a stronger demonstration of backend systems design. Both versions are left in the repo:
- `apps/server` → **legacy Node/Express server** (kept for posterity).  
- `apps/go-server` → **current Go-based implementation** with database persistence, migrations, and daily challenges.

---

## Implemented Features

### Task 1 – Normal Wordle
- 5-letter word guessing game.  
- Configurable **max rounds** and **word lists**.  
- Classic scoring: **hit / present / miss**.  
- React-based frontend with keyboard + animation feedback.  

### Task 2 – Server/Client Wordle
- Client never sees the answer until win/lose.  
- Server validates guesses against allowed dictionary.  
- REST API endpoints (`/game/new`, `/game/guess`) with JSON responses.  
- **Persistent user accounts** (signup/login/logout) with JWT + cookie authentication.  
- Anonymous guest support via `anonymous_id`, later claimable when signing up.  

### Task 3 – Host Cheating Wordle
- Server maintains candidate word buckets instead of committing to one answer up front.  
- Implements **Absurdle-like tie-break rules**:  
  - Fewer hits rank lower than more hits.  
  - If equal, fewer presents rank lower.  
- Bucketing algorithm (`nextCheatingCandidates`) produces consistent, opaque behavior to the player.  

### Task 4 – Multi-player Wordle
- Designed **Daily Challenge mode**:  
  - Shared answer across all players each day.  
  - Leaderboard ranked by elapsed time and guesses.  
  - Enforces one attempt per user/day.  
- In-memory + DB-backed sessions for game state.  
- Future-facing multiplayer extensions (see roadmap below).  

---

## Bells & Whistles

- **Daily Challenge** with persistent results & leaderboard.  
- **Authentication & Profiles**:  
  - Sign-up / log-in with JWT cookies.  
  - Track games played, wins, streaks.  
  - Profile page with stats and recent games.  
- **Guest Mode** with migration to full account on signup.  
- **Frontend polish**:  
  - Toasts, banners, and animations.  
  - On-screen and physical keyboard support.  
  - Error shake animation on invalid words.  
- **Code Quality**:  
  - Strict typing with Zod schemas (`@wordle/protocol`).  
  - Shared game logic in a reusable `game-core` package.  
  - Well-documented Go modules (migrations, stores, game engine, words).  

---

## Refactor: Express → Go

I began with a Node/Express server (`apps/server/src/index.ts`) to validate requirements quickly as I am more familiar with that language.  

Once up and running, I migrated to **Go (`apps/go-server/`)**:
- Better concurrency and performance.  
- Strong type safety and minimal runtime dependencies.  
- Explicit database migrations (`apps/go-server/sql/*.sql`).  
- Structured logging with `zerolog`.  
- Clear modular organization:  
  - `internal/game/` – core game engine.  
  - `internal/daily/` – daily challenge + leaderboard.  
  - `internal/httpserver/` – routes, auth, middleware.  
  - `internal/store/` – pluggable game state stores.  
  - `internal/words/` – word list loading, scoring.  

This rewrite demonstrates ability to refactor while preserving functionality.

---

## Setup & Running

### Prerequisites
- Node.js 20+  
- Go 1.22+  
- SQLite (bundled with Go module)  

### Install dependencies

```
npm install
```

### Development (frontend + Go server concurrently)

```
npm run dev
```

This launches:
- Web client at `http://localhost:5173`
- Go server at `http://localhost:5175` (or :3000 by default in .env)

### Database

Migrations run automatically on startup:
- **users** – registered accounts.
- **games** – historical game records.
- **daily_results** – daily challenge entries.

### Docker

```
cd apps/go-server
docker build -t wordle/go-server:dev .
docker run --rm -p 5175:5175 --env-file .env wordle/go-server:dev
```

## Project Structure

```
apps/
  web/           # React frontend
  server/        # Legacy Node/Express backend (deprecated)
  go-server/     # Production Go backend
    internal/
      game/      # Core game logic
      daily/     # Daily challenge & leaderboard
      httpserver # Routes, auth, middleware
      store/     # Pluggable state stores
      words/     # Word lists + scoring
    sql/         # Database migrations
packages/
  game-core/     # Shared TS logic (scoring, cheating host)
  protocol/      # Shared schemas (Zod)
```

## Planned Improvements

These were identified during the project as natural next steps:
- **Challenges**
    Send direct challenges to friends via email, letting them play the same word or a custom puzzle.
- **Improved Profiles**
    Richer stats, badges/achievements, streak graphs, and game history visualizations.
- **Advanced Analytics**
    Aggregate play stats, distribution charts of guesses, heatmaps for popular starting words.
- **SWORDLE** (flagship multiplayer mode)
    Real-time, head-to-head speed competition.
    Each player solves their own Wordle in 30 second speed rounds.
    Progress pushes a “sword” toward the opponent’s side.
    First to win pushes the sword fully across = victory.
    Combines Wordle skill with arcade-like intensity.

## Trade-offs & Decisions

- **Express → Go rewrite:** favored long-term maintainability and system-level clarity.
- **SQLite with migrations:** simple, portable DB, good for prototyping.
- **Shared TS packages (game-core, protocol):** ensures consistent rules/scoring across client and server.
- **Guest vs Auth accounts:** chose to support both for accessibility, with a claim mechanism to merge progress.
- **Daily Challenge:** prioritized as a lightweight multiplayer alternative that demonstrates persistence, leaderboards, and fairness over the much heavier requirements of the planned **SWORDLE**.

## Measurement Criteria Coverage

- **Understanding of abstract problem:** Implemented all four assignment tasks + extensions.
- **Decision making:** Documented trade-offs (e.g., Express → Go, guest accounts).
- **Code quality:** Modular Go packages, typed TS, comprehensive documentation.
- **Documentation:** Code-level docstrings + this README.
- **Repository practice:** Clean commits, workspace organization, Dockerfile/Makefile.

## Demo

- Visit http://localhost:5173 after running npm run dev.
- Play classic Wordle, Daily Challenge, or Cheating Host.
- Sign up to track progress, streaks, and leaderboard placement.

## Conclusion

This project demonstrates:
- End-to-end design of a Wordle system.
- Clean frontend + backend separation.
- Ability to pivot tech stack mid-project while maintaining features.
- Delivery of core tasks and beyond with bells & whistles.
- Clear roadmap for advanced features like SWORDLE and enhanced profiles.