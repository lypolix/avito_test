CREATE TABLE pr_reviewers (
    pr_id VARCHAR(50) NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    user_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (pr_id, user_id)
);

CREATE INDEX idx_reviewer_user ON pr_reviewers(user_id);                        