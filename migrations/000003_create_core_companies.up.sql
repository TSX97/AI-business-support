CREATE TABLE core.companies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    industry VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);
