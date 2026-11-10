CREATE TABLE core.company_members (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    company_id INTEGER REFERENCES core.companies(id),
    role VARCHAR(50) NOT NULL,

    CONSTRAINT fk_company_members_user
        FOREIGN KEY (user_id)
            REFERENCES auth.users(id),

    CONSTRAINT uq_company_members_user_company
        UNIQUE (user_id, company_id)
);
