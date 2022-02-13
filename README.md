# Long running job queue

Priority Queue built on postgres. Main difference with other projects are

- API is exposed over REST so we are language agnostic w.r.t clients
- A transaction is not held during work, because we assume jobs to be long running. This means clients are responsible for sending work heartbeats, and ack/nacking.

# Usage

You are responsible for running the migrations against your postgres database.

# Development

Run the migrations

```bash
migrate -path src/adapters/postgres_store/migrations -database "pgx://lrjq:lrjq@localhost:5432/queue" up
```
