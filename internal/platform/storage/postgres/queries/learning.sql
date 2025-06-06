-- Learning system queries

-- =====================================================
-- LEARNING EVENTS QUERIES
-- =====================================================

-- name: CreateLearningEvent :one
INSERT INTO learning_events (
    user_id,
    event_type,
    context,
    input_data,
    output_data,
    outcome,
    confidence,
    feedback_score,
    learning_metadata,
    duration_ms,
    session_id,
    correlation_id
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetLearningEvents :many
SELECT * FROM learning_events
WHERE user_id = $1::uuid
  AND (event_type = ANY($2::text[]) OR $2 IS NULL)
  AND (session_id = $3 OR $3 IS NULL)
  AND created_at >= COALESCE($4, NOW() - INTERVAL '30 days')
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: GetLearningEventsByCorrelation :many
SELECT * FROM learning_events
WHERE correlation_id = $1::uuid
ORDER BY created_at ASC;

-- name: GetLearningEventStats :one
SELECT
    event_type,
    COUNT(*) as event_count,
    AVG(confidence) as avg_confidence,
    AVG(duration_ms) as avg_duration_ms,
    COUNT(*) FILTER (WHERE outcome = 'success') as success_count,
    COUNT(*) FILTER (WHERE outcome = 'failure') as failure_count
FROM learning_events
WHERE user_id = $1::uuid
  AND created_at >= COALESCE($2, NOW() - INTERVAL '7 days')
GROUP BY event_type;

-- name: UpdateLearningEventFeedback :one
UPDATE learning_events
SET feedback_score = $2,
    learning_metadata = learning_metadata || $3
WHERE id = $1
RETURNING *;

-- name: DeleteOldLearningEvents :exec
DELETE FROM learning_events
WHERE user_id = $1::uuid
  AND created_at < $2;

-- =====================================================
-- LEARNED PATTERNS QUERIES
-- =====================================================

-- name: CreateLearnedPattern :one
INSERT INTO learned_patterns (
    user_id,
    pattern_type,
    pattern_name,
    pattern_signature,
    pattern_data,
    confidence,
    occurrence_count
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7
) ON CONFLICT (user_id, pattern_type, pattern_signature)
DO UPDATE SET
    occurrence_count = learned_patterns.occurrence_count + 1,
    confidence = ($6 + learned_patterns.confidence) / 2,
    pattern_data = $5,
    last_observed = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetLearnedPatterns :many
SELECT * FROM learned_patterns
WHERE user_id = $1::uuid
  AND (pattern_type = ANY($2::text[]) OR $2 IS NULL)
  AND is_active = true
  AND confidence >= COALESCE($3, 0.0)
ORDER BY confidence DESC, last_observed DESC
LIMIT $4 OFFSET $5;

-- name: GetLearnedPattern :one
SELECT * FROM learned_patterns
WHERE user_id = $1::uuid 
  AND pattern_type = $2 
  AND pattern_signature = $3;

-- name: UpdatePatternOutcome :one
UPDATE learned_patterns
SET positive_outcomes = positive_outcomes + $2,
    negative_outcomes = negative_outcomes + $3,
    confidence = CASE 
        WHEN ($2 + $3) > 0 THEN 
            LEAST(1.0, positive_outcomes::float / (positive_outcomes + negative_outcomes + $2 + $3))
        ELSE confidence
    END,
    last_observed = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivatePattern :exec
UPDATE learned_patterns
SET is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: SearchPatternsByData :many
SELECT * FROM learned_patterns
WHERE user_id = $1::uuid
  AND pattern_data @> $2
  AND is_active = true
ORDER BY confidence DESC
LIMIT $3;

-- =====================================================
-- USER SKILLS QUERIES
-- =====================================================

-- name: CreateUserSkill :one
INSERT INTO user_skills (
    user_id,
    skill_category,
    skill_name,
    proficiency_level,
    experience_points
) VALUES (
    $1::uuid, $2, $3, $4, $5
) ON CONFLICT (user_id, skill_category, skill_name)
DO UPDATE SET
    proficiency_level = $4,
    experience_points = user_skills.experience_points + $5,
    updated_at = NOW()
RETURNING *;

-- name: GetUserSkills :many
SELECT * FROM user_skills
WHERE user_id = $1::uuid
  AND (skill_category = $2 OR $2 IS NULL)
ORDER BY proficiency_level DESC, experience_points DESC
LIMIT $3 OFFSET $4;

-- name: GetUserSkill :one
SELECT * FROM user_skills
WHERE user_id = $1::uuid 
  AND skill_category = $2 
  AND skill_name = $3;

-- name: UpdateSkillUsage :one
UPDATE user_skills
SET successful_uses = successful_uses + $2,
    total_uses = total_uses + $3,
    last_used = NOW(),
    proficiency_level = LEAST(1.0, proficiency_level + ($2::float / GREATEST(total_uses + $3, 1)) * 0.1),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: AddSkillLearningPoint :one
UPDATE user_skills
SET learning_curve = learning_curve || jsonb_build_object(
    'timestamp', EXTRACT(EPOCH FROM NOW()),
    'proficiency', $2,
    'context', $3
),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetSkillProgression :one
SELECT 
    skill_name,
    proficiency_level,
    experience_points,
    successful_uses,
    total_uses,
    CASE 
        WHEN total_uses > 0 THEN successful_uses::float / total_uses 
        ELSE 0 
    END as success_rate,
    learning_curve
FROM user_skills
WHERE user_id = $1::uuid 
  AND skill_category = $2 
  AND skill_name = $3;

-- name: GetTopSkills :many
SELECT 
    skill_category,
    skill_name,
    proficiency_level,
    experience_points,
    last_used
FROM user_skills
WHERE user_id = $1::uuid
ORDER BY proficiency_level DESC, experience_points DESC
LIMIT $2;

-- name: GetSkillsByProficiency :many
SELECT * FROM user_skills
WHERE user_id = $1::uuid
  AND proficiency_level >= $2
  AND proficiency_level <= $3
ORDER BY skill_category, skill_name;

-- name: UpdateSkillRelatedPatterns :one
UPDATE user_skills
SET related_patterns = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;