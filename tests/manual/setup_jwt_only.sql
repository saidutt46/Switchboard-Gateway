-- Disable API Key auth, enable JWT auth only (global)

-- Delete existing auth plugins
DELETE FROM plugins WHERE name = 'api-key-auth';
DELETE FROM plugins WHERE name = 'jwt-auth';

-- Add JWT auth globally
INSERT INTO plugins (
    name,
    scope,
    config,
    enabled,
    priority
)
VALUES (
    'jwt-auth',
    'global',
    json_build_object(
        'critical', false,
        'secret_key', 'my-test-secret-key-12345',
        'algorithm', 'HS256',
        'cookie_name', 'jwt_token',
        'header_name', 'Authorization',
        'issuer', 'test-gateway',
        'audience', 'test-api',
        'cache_ttl_seconds', 300,
        'bypass_paths', json_build_array('/health', '/ready', '/metrics')
    ),
    true,
    6
);

SELECT 'JWT auth enabled globally!' AS status;

SELECT id, name, scope, enabled, priority
FROM plugins
WHERE name IN ('api-key-auth', 'jwt-auth');