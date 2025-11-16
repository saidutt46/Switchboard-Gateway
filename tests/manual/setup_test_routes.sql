-- Setup routes for radix tree testing with go-httpbin
-- Run with: psql -U switchboard -d switchboard -f setup_test_routes.sql

-- Clean up first (if service exists, delete it and all its routes)
DELETE FROM routes WHERE service_id IN (SELECT id FROM services WHERE name = 'user-service');
DELETE FROM services WHERE name = 'user-service';

-- Create test service pointing to httpbin
INSERT INTO services (name, protocol, host, port, path, enabled)
VALUES ('user-service', 'http', 'localhost', 8081, NULL, true);

-- Create routes with different patterns to test radix tree

-- Route 1: Basic httpbin endpoints (exact matches)
INSERT INTO routes (service_id, name, paths, methods, enabled)
SELECT 
    id,
    'httpbin-basic',
    ARRAY['/get', '/post', '/put', '/delete', '/patch', '/anything'],
    ARRAY['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'],
    true
FROM services 
WHERE name = 'user-service';

-- Route 2: Status code endpoint (parameter extraction test)
INSERT INTO routes (service_id, name, paths, methods, enabled)
SELECT 
    id,
    'httpbin-status',
    ARRAY['/status/:code'],
    ARRAY['GET'],
    true
FROM services 
WHERE name = 'user-service';

-- Route 3: Delay endpoint (another parameter test)
INSERT INTO routes (service_id, name, paths, methods, enabled)
SELECT 
    id,
    'httpbin-delay',
    ARRAY['/delay/:seconds'],
    ARRAY['GET'],
    true
FROM services 
WHERE name = 'user-service';

-- Route 4: Headers endpoints (multiple exact matches)
INSERT INTO routes (service_id, name, paths, methods, enabled)
SELECT 
    id,
    'httpbin-headers',
    ARRAY['/headers', '/response-headers'],
    ARRAY['GET'],
    true
FROM services 
WHERE name = 'user-service';

-- Show the configuration
SELECT 
    s.name as service_name,
    CONCAT(s.protocol, '://', s.host, ':', s.port) as target,
    r.name as route_name,
    r.paths,
    array_length(r.paths, 1) as path_count
FROM services s
JOIN routes r ON s.id = r.service_id
WHERE s.name = 'user-service'
ORDER BY r.created_at;

-- Show summary for radix tree
SELECT 
    COUNT(DISTINCT r.id) as total_routes,
    SUM(array_length(r.paths, 1)) as total_paths_in_tree
FROM routes r
JOIN services s ON r.service_id = s.id
WHERE s.name = 'user-service' AND r.enabled = true;

-- Success message
SELECT '';
SELECT '=== Setup Complete ===';
SELECT 'Service: user-service -> http://localhost:8081';
SELECT 'Routes created with patterns for radix tree testing';
SELECT '';
SELECT 'Start httpbin with: docker run -d -p 8081:80 kennethreitz/httpbin';
SELECT 'Test with: curl http://localhost:8080/get';
SELECT '           curl http://localhost:8080/status/200';
SELECT '           curl http://localhost:8080/delay/1';
SELECT '';