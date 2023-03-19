CREATE TABLE todos
(
    id          VARCHAR(255) PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMP,
    updated_at  TIMESTAMP
)
