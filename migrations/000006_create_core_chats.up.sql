CREATE TABLE core.chats (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    company_id INTEGER REFERENCES core.companies(id),
    title VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT fk_chats_user
        FOREIGN KEY (user_id)
            REFERENCES auth.users(id)
);
