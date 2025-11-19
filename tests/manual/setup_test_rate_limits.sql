-- Test Rate Limit Configurations

-- Delete existing rate-limit plugins
DELETE FROM plugins WHERE name = 'rate-limit';

-- Insert global rate limit plugin
INSERT INTO plugins (
    name, 
    scope, 
    config, 
    enabled, 
    priority
)
VALUES (
    'rate-limit',
    'global',
    json_build_object(
        'critical', false,
        'algorithm', 'token-bucket',
        'limit', 10,
        'window', '1m',
        'identifier', 'auto',
        'redis_url', 'redis://localhost:6379/0',
        'key_prefix', 'rate_limit:',
        'headers', true,
        'response_code', 429,
        'response_message', 'Too many requests - please try again later'
    ),
    true,
    10
);

-- Verify insertion
SELECT 'Rate limit plugin created!' AS status;

-- Show the created plugin
SELECT id, name, scope, enabled, priority
FROM plugins
WHERE name = 'rate-limit';