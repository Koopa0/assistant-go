-- System events and event sourcing queries

-- =====================================================
-- SYSTEM EVENTS QUERIES
-- =====================================================

-- name: CreateSystemEvent :one
INSERT INTO system_events (
    event_type,
    aggregate_type,
    aggregate_id,
    user_id,
    event_data,
    event_metadata,
    event_version
) VALUES (
    $1, $2, $3::uuid, $4::uuid, $5, $6, $7
) RETURNING id, event_type, aggregate_type, aggregate_id, user_id, 
           event_data, event_metadata, event_version, created_at, 
           processed_at, processing_error;

-- name: GetSystemEvent :one
SELECT id, event_type, aggregate_type, aggregate_id, user_id,
       event_data, event_metadata, event_version, created_at,
       processed_at, processing_error
FROM system_events
WHERE id = $1;

-- name: GetEventsByAggregate :many
SELECT id, event_type, aggregate_type, aggregate_id, user_id,
       event_data, event_metadata, event_version, created_at,
       processed_at, processing_error
FROM system_events
WHERE aggregate_type = $1
  AND aggregate_id = $2::uuid
ORDER BY event_version ASC, created_at ASC;

-- name: GetEventsByType :many
SELECT id, event_type, aggregate_type, aggregate_id, user_id,
       event_data, event_metadata, event_version, created_at,
       processed_at, processing_error
FROM system_events
WHERE event_type = $1
  AND (user_id = $2::uuid OR $2 IS NULL)
  AND created_at >= COALESCE($3, NOW() - INTERVAL '30 days')
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetUnprocessedEvents :many
SELECT id, event_type, aggregate_type, aggregate_id, user_id,
       event_data, event_metadata, event_version, created_at,
       processed_at, processing_error
FROM system_events
WHERE processed_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkEventProcessed :one
UPDATE system_events
SET processed_at = NOW()
WHERE id = $1
RETURNING id, event_type, aggregate_type, aggregate_id, user_id,
          event_data, event_metadata, event_version, created_at,
          processed_at, processing_error;

-- name: MarkEventFailed :one
UPDATE system_events
SET processing_error = $2
WHERE id = $1
RETURNING id, event_type, aggregate_type, aggregate_id, user_id,
          event_data, event_metadata, event_version, created_at,
          processed_at, processing_error;

-- name: GetEventStatistics :one
SELECT 
    event_type,
    COUNT(*) as total_events,
    COUNT(*) FILTER (WHERE processed_at IS NOT NULL) as processed_events,
    COUNT(*) FILTER (WHERE processing_error IS NOT NULL) as failed_events,
    MAX(created_at) as latest_event,
    MIN(created_at) as earliest_event
FROM system_events
WHERE (user_id = $1::uuid OR $1 IS NULL)
  AND created_at >= COALESCE($2, NOW() - INTERVAL '7 days')
GROUP BY event_type
ORDER BY total_events DESC;

-- name: GetRecentEvents :many
SELECT id, event_type, aggregate_type, aggregate_id, user_id,
       event_data, event_metadata, event_version, created_at,
       processed_at, processing_error
FROM system_events
WHERE (user_id = $1::uuid OR $1 IS NULL)
  AND created_at >= COALESCE($2, NOW() - INTERVAL '24 hours')
ORDER BY created_at DESC
LIMIT $3;

-- name: DeleteOldEvents :exec
DELETE FROM system_events
WHERE created_at < $1
  AND processed_at IS NOT NULL;

-- =====================================================
-- EVENT PROJECTIONS QUERIES
-- =====================================================

-- name: CreateEventProjection :one
INSERT INTO event_projections (
    projection_name,
    projection_state
) VALUES (
    $1, $2
) ON CONFLICT (projection_name)
DO UPDATE SET
    projection_state = $2,
    updated_at = NOW()
RETURNING id, projection_name, last_processed_event_id, last_processed_at,
          projection_state, error_count, last_error, created_at, updated_at;

-- name: GetEventProjection :one
SELECT id, projection_name, last_processed_event_id, last_processed_at,
       projection_state, error_count, last_error, created_at, updated_at
FROM event_projections
WHERE projection_name = $1;

-- name: GetAllEventProjections :many
SELECT id, projection_name, last_processed_event_id, last_processed_at,
       projection_state, error_count, last_error, created_at, updated_at
FROM event_projections
ORDER BY projection_name;

-- name: UpdateProjectionProgress :one
UPDATE event_projections
SET last_processed_event_id = $2::uuid,
    last_processed_at = NOW(),
    error_count = 0,
    last_error = NULL,
    updated_at = NOW()
WHERE projection_name = $1
RETURNING id, projection_name, last_processed_event_id, last_processed_at,
          projection_state, error_count, last_error, created_at, updated_at;

-- name: RecordProjectionError :one
UPDATE event_projections
SET error_count = error_count + 1,
    last_error = $2,
    updated_at = NOW()
WHERE projection_name = $1
RETURNING id, projection_name, last_processed_event_id, last_processed_at,
          projection_state, error_count, last_error, created_at, updated_at;

-- name: ResetProjection :one
UPDATE event_projections
SET last_processed_event_id = NULL,
    last_processed_at = NULL,
    projection_state = '{}'::jsonb,
    error_count = 0,
    last_error = NULL,
    updated_at = NOW()
WHERE projection_name = $1
RETURNING id, projection_name, last_processed_event_id, last_processed_at,
          projection_state, error_count, last_error, created_at, updated_at;

-- name: GetProjectionStatus :one
SELECT 
    ep.projection_name,
    ep.last_processed_at,
    ep.error_count,
    ep.last_error,
    COUNT(se.id) as pending_events
FROM event_projections ep
LEFT JOIN system_events se ON se.id > COALESCE(ep.last_processed_event_id, '00000000-0000-0000-0000-000000000000'::uuid)
  AND se.processed_at IS NOT NULL
WHERE ep.projection_name = $1
GROUP BY ep.projection_name, ep.last_processed_at, ep.error_count, ep.last_error;