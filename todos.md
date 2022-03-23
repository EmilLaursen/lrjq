# TODOs

## DELIVERABLE 1

- [x] REMOVE HEADERS AGAIN
- [x] HOW IS TEST COVERAGE
- [x] CONFIGURATION from environment
- [x] REQUEUE FAILED goroutine
- [x] MOVE DONE goroutine
- [x] Openapi validation

## DELIVERABLE 2

- [x] Openapi spec: enqueue returns full msg. add model
- [x] Openapi, add links

- [x] Schemathesis test setup!
- [x] Msg is a byte array, not json
- [x] Add DB schema to tables
- [ ] remove openapi chi stuff
- [ ] Tests omg...
  - [x] If work times out (no heartbeat in time), any subsequent ack/nack fails!
  - nack after ack (or vice versa) fails
  - concurrency test
- [x] add DIRENV with path_add to binary folder
- [ ] maybe delete done_jobs table, or make it configurable. Maybe remove payload?
- [x] fix bug in dequeue - need maxtries guard
- [ ] nack, requeue openapi.
- [ ] tls certificates from env
- [ ] refactor json response stuff?
- [ ] BULK OPS
- [ ] NACK+ACK
- [ ] README
- [ ] Query/read deadletters
- [ ] CONFIGURABLE DB SCHEMA?
- [ ] ADD INDEX FOR DEQUEUE? `created_at` ? maybe `queue_id`
- [ ] HEARTBEAT SHIT (what does this mean)

- [-] fix openapi validation, path item not found?
  - pivit: use legacy router, remove chi router

## DELIVERABLE 3

- [ ] PLZ build system?
- [ ] CONFIGURABLE SCHEMA
- [ ] PROMETHEUS SHIT?
