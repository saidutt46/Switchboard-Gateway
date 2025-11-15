-- Setup routes for go-httpbin testing
-- Use localhost because gateway runs on host, not in Docker
UPDATE services 
SET host = 'localhost',  -- Use localhost, not demo-backend
    port = 8081,         -- Use host-mapped port
    protocol = 'http',
    path = NULL
WHERE name = 'user-service';

DELETE FROM routes WHERE service_id IN (SELECT id FROM services WHERE name = 'user-service');

INSERT INTO routes (service_id, name, paths, methods, enabled)
SELECT 
    id,
    'httpbin-endpoints',
    ARRAY['/get', '/post', '/put', '/delete', '/patch', '/anything', '/headers', '/status/:code'],
    ARRAY['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'],
    true
FROM services 
WHERE name = 'user-service';

-- Show the configuration
SELECT 
    s.name as service_name,
    CONCAT(s.protocol, '://', s.host, ':', s.port) as target,
    r.name as route_name,
    r.paths
FROM services s
JOIN routes r ON s.id = r.service_id
WHERE s.name = 'user-service';