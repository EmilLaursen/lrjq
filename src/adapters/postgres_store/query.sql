-- name: Enqueue :one
INSERT INTO jobs (payload, priority, queue_id) VALUES
(pggen.arg('payload'), pggen.arg('priority'), pggen.arg('queueID'))
RETURNING *;

-- name: Dequeue :one
WITH PEEK AS (
     SELECT id as peek_id
     FROM jobs
     WHERE
        status = 'ready' and
        queue_id = pggen.arg('queueID')
    ORDER BY priority, created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE jobs
SET
  started_at = now(),
  last_heartbeat = now(),
  status = 'started',
  tries = jobs.tries + 1,
  work_signature = uuid_generate_v4()
FROM PEEK
WHERE
  jobs.id = peek_id
RETURNING *;

-- name: ReportDone :exec
WITH moved_row AS (
     DELETE FROM jobs
     WHERE
        id = pggen.arg('id') AND
        work_signature = pggen.arg('workSignature') AND
        status = 'started'
     RETURNING *
)
INSERT INTO DONE_JOBS (id, queue_id, payload, created_at, last_heartbeat, done_at, tries, priority)
SELECT
   id,
   queue_id,
   payload,
   created_at,
   last_heartbeat,
   now(),
   tries,
   priority
FROM moved_row;

-- name: RequeueFailed :exec
UPDATE jobs
SET
  status = 'ready',
  started_at = null,
  last_heartbeat = null,
  work_signature = null
WHERE
  status = 'started' AND last_heartbeat <= now() - pggen.arg('deadline')::interval;


-- name: DeleteDeadLetters :exec
WITH DEAD_JOBS AS (
     DELETE FROM jobs
     WHERE tries >= pggen.arg('maxTries')
     RETURNING *
)
INSERT INTO dead_letters (id, queue_id, payload, created_at, last_heartbeat, done_at, tries, priority)
SELECT
  id,
  queue_id,
  payload,
  created_at,
  last_heartbeat,
  done_at,
  tries,
  priority
FROM DEAD_JOBS;

-- name: SendHeartBeat :exec
UPDATE jobs SET
  last_heartbeat = now()
WHERE
  id = pggen.arg('id') and work_signature = pggen.arg('workSignature') and status = 'started';
