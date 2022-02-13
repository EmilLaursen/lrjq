BEGIN;

DROP EXTENSION IF EXISTS "uuid-ossp";
DROP TABLE IF EXISTS queue.jobs;
DROP TABLE IF EXISTS queue.done_jobs;
DROP TABLE IF EXISTS queue.dead_letters;
DROP TYPE IF EXISTS queue.job_status;
DROP SCHEMA IF EXISTS queue;

COMMIT;
