BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE type job_status AS ENUM('ready', 'started', 'done');

CREATE TABLE IF NOT EXISTS JOBS(
                   id serial PRIMARY KEY,
                   queue_id text NOT NULL,
                   payload jsonb NOT NULL,
                   work_signature uuid,
                   created_at timestamptz NOT NULL DEFAULT NOW(),
                   last_heartbeat timestamptz,
                   started_at timestamptz,
                   done_at timestamptz,
                   tries integer NOT NULL DEFAULT 0,
                   priority integer NOT NULL DEFAULT 0,
                   status job_status DEFAULT 'ready' );

CREATE TABLE IF NOT EXISTS DONE_JOBS(
                   id serial PRIMARY KEY,
                   queue_id text NOT NULL,
                   payload jsonb NOT NULL,
                   created_at timestamptz NOT NULL,
                   last_heartbeat timestamptz,
                   started_at timestamptz,
                   done_at timestamptz NOT NULL DEFAULT NOW(),
                   tries integer NOT NULL DEFAULT 0,
                   priority integer NOT NULL DEFAULT 0);


CREATE TABLE IF NOT EXISTS DEAD_LETTERS(
                   id serial PRIMARY KEY,
                   queue_id text NOT NULL,
                   payload jsonb NOT NULL,
                   created_at timestamptz NOT NULL,
                   last_heartbeat timestamptz,
                   started_at timestamptz,
                   done_at timestamptz,
                   tries integer NOT NULL DEFAULT 0,
                   priority integer NOT NULL DEFAULT 0);
COMMIT;
