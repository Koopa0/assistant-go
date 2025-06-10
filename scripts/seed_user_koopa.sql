-- Seed user for Koopa
-- Password: KoopaAssistant2024!

-- Insert Koopa user with hashed password
-- Seed user: Koopa
-- Password: KoopaAssistant2024!
INSERT INTO users (
    id,
    username,
    email,
    password_hash,
    full_name,
    is_active,
    is_verified,
    role,
    preferences,
    created_at,
    updated_at
) VALUES (
    'a0000000-0000-4000-8000-000000000001'::uuid,
    'koopa',
    'koopa@assistant.local',
    '$2a$10$Ib62kjkKz3qD6GAcycBF5evEpr1p5Vm7ZVXDcvvwpqISQtIFnWzM.',
    'Koopa',
    true,
    true,
    'admin',
    '{"theme": "dark", "language": "zh-TW", "defaultProgrammingLanguage": "Go"}'::jsonb,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    is_active = EXCLUDED.is_active,
    is_verified = EXCLUDED.is_verified,
    role = EXCLUDED.role,
    updated_at = CURRENT_TIMESTAMP;
