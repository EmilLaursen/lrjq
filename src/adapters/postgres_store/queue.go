package postgres_store

//go:generate pggen gen go --output-dir gen/ -query-glob query.sql -schema-glob migrations/*.up.sql
