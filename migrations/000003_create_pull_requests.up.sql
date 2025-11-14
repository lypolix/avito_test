CREATE TABLE pull_requests (
    pull_request_id VARCHAR(50) PRIMARY KEY,  
    pull_request_name VARCHAR(500) NOT NULL,  
    author_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
    status VARCHAR(20) DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'MERGED')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    merged_at TIMESTAMP NULL
);

CREATE INDEX idx_pr_author ON pull_requests(author_id);
CREATE INDEX idx_pr_status ON pull_requests(status);