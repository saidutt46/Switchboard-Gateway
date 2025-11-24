-- Test API Key Authentication Setup

-- Delete existing api-key-auth plugins first
DELETE FROM plugins WHERE name = 'api-key-auth';

-- Delete and recreate test consumer
DELETE FROM consumers WHERE id = '550e8400-e29b-41d4-a716-446655440001';

INSERT INTO consumers (id, username, email, custom_id, metadata, created_at, updated_at)
VALUES (
    '550e8400-e29b-41d4-a716-446655440001',
    'test-partner',
    'test@example.com',
    'partner-001',
    CAST('{"tier": "premium"}' AS jsonb),
    NOW(),
    NOW()
);

-- Create API key for test consumer
-- Plaintext: test-key-12345
-- Hash: 953a6f3acb148f7d0492a99ed5ce98dd442326f6438b39625fd5c85efa7f6f21
INSERT INTO api_keys (id, consumer_id, key_hash, name, enabled, created_at)
VALUES (
    '660e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440001',
    '953a6f3acb148f7d0492a99ed5ce98dd442326f6438b39625fd5c85efa7f6f21',
    'Test API Key',
    true,
    NOW()
);

-- Insert API key authentication plugin (global)
INSERT INTO plugins (
    name,
    scope,
    config,
    enabled,
    priority
)
VALUES (
    'api-key-auth',
    'global',
    json_build_object(
        'critical', false,
        'key_names', json_build_array('X-API-Key', 'apikey'),
        'key_in_query', true,
        'key_in_header', true,
        'hide_credentials', true,
        'cache_ttl_seconds', 300,
        'anonymous_allowed', false,
        'bypass_paths', json_build_array('/health', '/ready', '/metrics')
    ),
    true,
    5
);

-- Verify insertion
SELECT 'API key auth plugin created!' AS status;

-- Show created plugin
SELECT id, name, scope, enabled, priority
FROM plugins
WHERE name = 'api-key-auth';

-- Show created consumer
SELECT id, username, email
FROM consumers
WHERE id = '550e8400-e29b-41d4-a716-446655440001';

-- Show created API key
SELECT id, name, enabled
FROM api_keys
WHERE consumer_id = '550e8400-e29b-41d4-a716-446655440001';