CREATE TABLE core.business (
    id SERIAL PRIMARY KEY,
    company_id INTEGER REFERENCES core.companies(id),
    data_type VARCHAR(100) NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
