# Long running job queue

Priority Queue built on postgres. Main difference with other projects are

- API is exposed over REST so we are language agnostic w.r.t clients
- A transaction is not held during work, because we assume jobs to be long running. This means clients are responsible for sending work heartbeats, and ack/nacking.

# Usage

You are responsible for running the migrations against your postgres database.

# Development

Run the migrations

```bash
migrate -path src/adapters/postgres_store/migrations -database "pgx://lrjq:lrjq@localhost:5432/testdb" up

migrate -path src/adapters/postgres_store/migrations -database "pgx://lrjq:lrjq@localhost:5432/queue" up
```

# Run schemathesis fuzz tests

Schemathesis can not handle multifile openapi definitions. Therefore we have to bundle the files first:

```bash
openapi-cli bundle openapi/openapi.yaml > tmp.yaml

schemathesis run --workers 8 --hypothesis-max-examples 1000 --stateful=links --show-errors-tracebacks --checks all --validate-schema false --base-url "http://localhost:8796" tmp.yaml
```

# Aliases

TODO: schemathesis needs pwd mount for filebased test to work

```bash
alias schemathesis='docker run --rm --net host -it schemathesis/schemathesis:stable'
alias openapi-cli='docker run --rm -it --net host -v $PWD:/spec redocly/openapi-cli'
```
