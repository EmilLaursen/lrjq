BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA if not exists queue;

CREATE type job_status AS ENUM('ready', 'started', 'done');

CREATE TABLE IF NOT EXISTS queue.jobs(
                   id serial PRIMARY KEY,
                   queue_id text NOT NULL,
                   payload bytea NOT NULL,
                   work_signature uuid,
                   created_at timestamptz NOT NULL DEFAULT NOW(),
                   last_heartbeat timestamptz,
                   started_at timestamptz,
                   done_at timestamptz,
                   tries integer NOT NULL DEFAULT 0,
                   priority integer NOT NULL DEFAULT 0,
                   status job_status DEFAULT 'ready' );

CREATE TABLE IF NOT EXISTS queue.done_jobs(
                   id serial PRIMARY KEY,
                   queue_id text NOT NULL,
                   payload bytea NOT NULL,
                   created_at timestamptz NOT NULL,
                   last_heartbeat timestamptz,
                   started_at timestamptz,
                   done_at timestamptz NOT NULL DEFAULT NOW(),
                   tries integer NOT NULL DEFAULT 0,
                   priority integer NOT NULL DEFAULT 0);


CREATE TABLE IF NOT EXISTS queue.dead_letters(
                   id serial PRIMARY KEY,
                   queue_id text NOT NULL,
                   payload bytea NOT NULL,
                   created_at timestamptz NOT NULL,
                   last_heartbeat timestamptz,
                   started_at timestamptz,
                   done_at timestamptz,
                   tries integer NOT NULL DEFAULT 0,
                   priority integer NOT NULL DEFAULT 0,
                   dead_lettered_at timestamptz NOT NULL DEFAULT now()
);
COMMIT;
