# Enma

An AI-enhanced journaling application. Write journal entries over time and use an AI chat interface to reflect on your history, track goals automatically, and get insights on personal growth — all through natural conversation.

## Stack

- **Backend** — Go + Fiber
- **Frontend** — Tauri + Svelte
- **Database** — PostgreSQL (pgx), Cassandra (chat messages)
- **AI** — OpenAI (chat + embeddings)
- **Auth** — JWT with rotating refresh tokens, Argon2id password hashing

## Prerequisites

- [Go](https://go.dev) 1.26+
- [Docker](https://www.docker.com)
- [Just](https://github.com/casey/just) — `brew install just`

## Backend setup

```bash
cd backend
cp .env.example .env   # fill in JWT_SECRET
just up                # start postgres + cassandra, run migrations
just run               # start the server on :8080
```

## Just recipes

Run from the project root:

```
just refs-clone                  clone all reference repos listed in references/repos.txt
just refs-pull                   pull latest changes in all cloned reference repos
```

Run from `backend/`:

```
just up                          start postgres + cassandra and run migrations
just down                        stop and remove all containers
just run                         run the server
just build                       build binary to bin/server
just migrate-up                  apply all pending migrations
just migrate-down                roll back the last migration
just migrate-create name=<name>  create a new migration file pair
just psql                        open a psql shell
just cqlsh                       open a cqlsh shell
```
