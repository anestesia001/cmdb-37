CREATE TABLE IF NOT EXISTS servers (
    id SERIAL PRIMARY KEY,
    hostname VARCHAR(255) UNIQUE NOT NULL,
    ip_address INET UNIQUE NOT NULL,
    os VARCHAR(100) NOT NULL,
    environment VARCHAR(20) NOT NULL
        CHECK (environment IN ('dev', 'stage', 'prod')),
    status VARCHAR(20) NOT NULL
        CHECK (status IN ('running', 'stopped', 'maintenance')),
    cpu INTEGER NOT NULL CHECK (cpu > 0),
    ram INTEGER NOT NULL CHECK (ram > 0),
    team VARCHAR(100) NOT NULL
);

INSERT INTO servers (
    hostname, ip_address, os, environment,
    status, cpu, ram, team
)
VALUES
('web-prod-01', '192.168.10.11', 'Ubuntu 24.04',
 'prod', 'running', 4, 8192, 'devops'),

('web-prod-02', '192.168.10.12', 'Ubuntu 24.04',
 'prod', 'running', 4, 8192, 'devops'),

('db-prod-01', '192.168.10.20', 'Debian 12',
 'prod', 'running', 8, 16384, 'dba'),

('app-stage-01', '192.168.20.11', 'Ubuntu 22.04',
 'stage', 'stopped', 2, 4096, 'backend'),

('app-dev-01', '192.168.30.11', 'Debian 12',
 'dev', 'maintenance', 2, 2048, 'backend');
