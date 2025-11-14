CREATE TABLE users (
    user_id VARCHAR(50) PRIMARY KEY,        
    username VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL REFERENCES teams(team_name),  
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  
);

CREATE INDEX idx_users_team_name ON users(team_name);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;
CREATE INDEX idx_users_team_active ON users(team_name, is_active);