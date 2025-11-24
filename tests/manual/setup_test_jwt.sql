-- Test JWT Authentication Setup

-- Delete existing jwt-auth plugins first
DELETE FROM plugins WHERE name = 'jwt-auth';

-- Insert JWT authentication plugin (global)
-- Secret key: "my-test-secret-key-12345" (for testing only!)
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

-- Verify insertion
SELECT 'JWT auth plugin created!' AS status;

-- Show created plugin
SELECT id, name, scope, enabled, priority
FROM plugins
WHERE name = 'jwt-auth';