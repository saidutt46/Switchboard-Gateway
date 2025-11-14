-- ============================================================================
-- Switchboard Gateway - Database Schema
-- Version: 1.0
-- Description: Complete schema for API Gateway configuration
-- ============================================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- TABLE: services
-- Purpose: Backend microservices/systems that the gateway proxies to
-- ============================================================================
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    
    -- Connection details
    protocol VARCHAR(10) NOT NULL CHECK (protocol IN ('http', 'https', 'grpc')),
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 80,
    path VARCHAR(255),
    
    -- Timeouts (milliseconds)
    connect_timeout_ms INTEGER DEFAULT 5000,
    read_timeout_ms INTEGER DEFAULT 60000,
    write_timeout_ms INTEGER DEFAULT 60000,
    retries INTEGER DEFAULT 0,
    
    -- Load balancing
    load_balancer_type VARCHAR(50) DEFAULT 'round-robin' 
        CHECK (load_balancer_type IN ('round-robin', 'least-connections', 'weighted', 'ip-hash')),
    
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index for fast lookups by name
CREATE INDEX idx_services_name ON services(name);
CREATE INDEX idx_services_enabled ON services(enabled);

-- ============================================================================
-- TABLE: service_targets
-- Purpose: Multiple backend instances for load balancing
-- ============================================================================
CREATE TABLE service_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    target VARCHAR(255) NOT NULL, -- Format: "host:port"
    weight INTEGER DEFAULT 100,
    health_check_path VARCHAR(255) DEFAULT '/health',
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(service_id, target)
);

-- Index for fast lookups by service
CREATE INDEX idx_service_targets_service_id ON service_targets(service_id);
CREATE INDEX idx_service_targets_enabled ON service_targets(enabled);

-- ============================================================================
-- TABLE: routes
-- Purpose: Maps incoming requests to services based on path/method/host
-- ============================================================================
CREATE TABLE routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    name VARCHAR(100),
    
    -- Matching criteria
    hosts TEXT[], -- Array of hostnames (e.g., ["api.example.com", "*.example.com"])
    paths TEXT[] NOT NULL, -- Array of path patterns (e.g., ["/api/users", "/api/users/:id"])
    methods TEXT[] DEFAULT ARRAY['GET','POST','PUT','DELETE','PATCH','OPTIONS','HEAD'],
    
    -- Path handling
    strip_path BOOLEAN DEFAULT false,
    preserve_host BOOLEAN DEFAULT false,
    
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for route matching performance
CREATE INDEX idx_routes_service_id ON routes(service_id);
CREATE INDEX idx_routes_enabled ON routes(enabled);
CREATE INDEX idx_routes_paths ON routes USING GIN (paths);
CREATE INDEX idx_routes_methods ON routes USING GIN (methods);

-- ============================================================================
-- TABLE: consumers
-- Purpose: API clients (applications/services calling the gateway)
-- ============================================================================
CREATE TABLE consumers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255),
    custom_id VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for consumer lookups
CREATE INDEX idx_consumers_username ON consumers(username);
CREATE INDEX idx_consumers_custom_id ON consumers(custom_id);

-- ============================================================================
-- TABLE: api_keys
-- Purpose: Authentication credentials for consumers
-- Note: Stores SHA256 hash, NEVER plaintext keys
-- ============================================================================
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consumer_id UUID NOT NULL REFERENCES consumers(id) ON DELETE CASCADE,
    key_hash VARCHAR(64) UNIQUE NOT NULL, -- SHA256 hash (64 hex chars)
    name VARCHAR(100),
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP
);

-- Indexes for authentication performance (critical path!)
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_consumer_id ON api_keys(consumer_id);
CREATE INDEX idx_api_keys_enabled ON api_keys(enabled);

-- ============================================================================
-- TABLE: plugins
-- Purpose: Modular functionality (auth, rate limiting, caching, etc.)
-- ============================================================================
CREATE TABLE plugins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL,
    scope VARCHAR(20) NOT NULL CHECK (scope IN ('global', 'service', 'route', 'consumer')),
    
    -- Foreign keys (only one should be set based on scope)
    service_id UUID REFERENCES services(id) ON DELETE CASCADE,
    route_id UUID REFERENCES routes(id) ON DELETE CASCADE,
    consumer_id UUID REFERENCES consumers(id) ON DELETE CASCADE,
    
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 100,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Constraint: Ensure only appropriate FK is set based on scope
    CONSTRAINT plugins_scope_fk_check CHECK (
        (scope = 'global' AND service_id IS NULL AND route_id IS NULL AND consumer_id IS NULL) OR
        (scope = 'service' AND service_id IS NOT NULL AND route_id IS NULL AND consumer_id IS NULL) OR
        (scope = 'route' AND route_id IS NOT NULL AND service_id IS NULL AND consumer_id IS NULL) OR
        (scope = 'consumer' AND consumer_id IS NOT NULL AND service_id IS NULL AND route_id IS NULL)
    )
);

-- Indexes for plugin lookups (critical for request processing!)
CREATE INDEX idx_plugins_scope ON plugins(scope);
CREATE INDEX idx_plugins_service_id ON plugins(service_id);
CREATE INDEX idx_plugins_route_id ON plugins(route_id);
CREATE INDEX idx_plugins_consumer_id ON plugins(consumer_id);
CREATE INDEX idx_plugins_enabled ON plugins(enabled);
CREATE INDEX idx_plugins_priority ON plugins(priority);

-- ============================================================================
-- TRIGGERS: Auto-update timestamps
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_services_updated_at BEFORE UPDATE ON services
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_routes_updated_at BEFORE UPDATE ON routes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_consumers_updated_at BEFORE UPDATE ON consumers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_plugins_updated_at BEFORE UPDATE ON plugins
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SAMPLE DATA (for development/testing)
-- ============================================================================

-- Sample Service: User Service
INSERT INTO services (name, protocol, host, port) VALUES
('user-service', 'http', 'demo-backend', 80);

-- Sample Service Targets (load balancing)
INSERT INTO service_targets (service_id, target, weight) VALUES
((SELECT id FROM services WHERE name = 'user-service'), 'demo-backend:80', 100);

-- Sample Route: User API
INSERT INTO routes (service_id, name, paths, methods) VALUES
((SELECT id FROM services WHERE name = 'user-service'), 
 'user-api', 
 ARRAY['/api/users', '/api/users/:id'],
 ARRAY['GET', 'POST', 'PUT', 'DELETE']);

-- Sample Consumer: Test App
INSERT INTO consumers (username, email) VALUES
('test-app', 'test@switchboard.dev');

-- Sample API Key (hashed: "test-key-12345")
-- Key: test-key-12345
-- SHA256: 5d5be9d12e23f8b8c3e3f4c36d3a3c3b3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c
INSERT INTO api_keys (consumer_id, key_hash, name) VALUES
((SELECT id FROM consumers WHERE username = 'test-app'),
 '5d5be9d12e23f8b8c3e3f4c36d3a3c3b3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c',
 'Test API Key');

-- Sample Plugin: Global Rate Limit
INSERT INTO plugins (name, scope, config, priority) VALUES
('rate-limit', 'global', '{"limit": 1000, "window": "1m", "algorithm": "sliding-window"}', 20);

-- ============================================================================
-- VIEWS (for analytics/reporting)
-- ============================================================================

-- View: Active routes with service info
CREATE VIEW v_active_routes AS
SELECT 
    r.id as route_id,
    r.name as route_name,
    r.paths,
    r.methods,
    s.id as service_id,
    s.name as service_name,
    s.protocol,
    s.host,
    s.port
FROM routes r
JOIN services s ON r.service_id = s.id
WHERE r.enabled = true AND s.enabled = true;

-- View: Consumer API key status
CREATE VIEW v_consumer_keys AS
SELECT 
    c.id as consumer_id,
    c.username,
    c.email,
    k.id as key_id,
    k.name as key_name,
    k.enabled as key_enabled,
    k.created_at as key_created_at,
    k.last_used_at as key_last_used_at,
    k.expires_at as key_expires_at
FROM consumers c
LEFT JOIN api_keys k ON c.id = k.consumer_id;

-- ============================================================================
-- FUNCTIONS (utility functions)
-- ============================================================================

-- Function: Get all plugins for a route (includes global, service, and route-specific)
CREATE OR REPLACE FUNCTION get_route_plugins(p_route_id UUID)
RETURNS TABLE (
    plugin_id UUID,
    plugin_name VARCHAR,
    plugin_config JSONB,
    plugin_priority INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.id,
        p.name,
        p.config,
        p.priority
    FROM plugins p
    LEFT JOIN routes r ON p.route_id = r.id
    WHERE p.enabled = true
      AND (
          p.scope = 'global'
          OR (p.scope = 'service' AND p.service_id = (SELECT service_id FROM routes WHERE id = p_route_id))
          OR (p.scope = 'route' AND p.route_id = p_route_id)
      )
    ORDER BY p.priority ASC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Additional composite indexes for common queries
CREATE INDEX idx_routes_service_enabled ON routes(service_id, enabled);
CREATE INDEX idx_plugins_route_enabled ON plugins(route_id, enabled) WHERE route_id IS NOT NULL;
CREATE INDEX idx_plugins_service_enabled ON plugins(service_id, enabled) WHERE service_id IS NOT NULL;

-- ============================================================================
-- COMPLETION
-- ============================================================================
-- Schema created successfully!
-- 
-- Tables created:
--   - services (6 rows with sample data)
--   - service_targets
--   - routes
--   - consumers
--   - api_keys
--   - plugins
-- 
-- Views created:
--   - v_active_routes
--   - v_consumer_keys
--
-- Functions created:
--   - get_route_plugins(route_id)
--   - update_updated_at_column()
-- ============================================================================