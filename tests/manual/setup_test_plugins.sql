-- Setup test plugins for Phase 7 testing
-- Run with: docker exec -i switchboard-postgres psql -U switchboard -d switchboard < tests/manual/setup_test_plugins.sql

-- Clean up existing plugins first
DELETE FROM plugins;

-- ============================================================================
-- Plugin 1: Request Logger (Global Scope)
-- ============================================================================
-- Priority 1 - Runs first in the chain
-- Logs all requests except health checks
INSERT INTO plugins (name, scope, config, priority, enabled)
VALUES (
  'request-logger',
  'global',
  '{
    "critical": false,
    "log_headers": true,
    "log_query_params": true,
    "excluded_paths": ["/health", "/ready"],
    "max_body_log_size": 0
  }',
  1,
  true
);

-- ============================================================================
-- Plugin 2: CORS (Global Scope)
-- ============================================================================
-- Priority 5 - Runs after logging
-- Allows all origins for development
INSERT INTO plugins (name, scope, config, priority, enabled)
VALUES (
  'cors',
  'global',
  '{
    "critical": false,
    "allowed_origins": ["*"],
    "allowed_methods": ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"],
    "allowed_headers": ["Content-Type", "Authorization", "X-API-Key"],
    "exposed_headers": ["X-Request-ID"],
    "allow_credentials": false,
    "max_age": 86400
  }',
  5,
  true
);

-- ============================================================================
-- Optional: Service-Specific CORS Override
-- ============================================================================
-- Uncomment this to test service-scoped plugins
-- This will override global CORS for user-service only

-- INSERT INTO plugins (name, scope, service_id, config, priority, enabled)
-- SELECT 
--   'cors',
--   'service',
--   s.id,
--   '{
--     "critical": false,
--     "allowed_origins": ["https://example.com"],
--     "allowed_methods": ["GET", "POST"],
--     "allow_credentials": true,
--     "max_age": 3600
--   }'::jsonb,
--   5,
--   true
-- FROM services s
-- WHERE s.name = 'user-service';

-- Show what we created
SELECT 
  id,
  name,
  scope,
  priority,
  enabled,
  config
FROM plugins
ORDER BY priority;

-- Show plugin count
SELECT 
  COUNT(*) as total_plugins,
  SUM(CASE WHEN enabled = true THEN 1 ELSE 0 END) as enabled_plugins
FROM plugins;