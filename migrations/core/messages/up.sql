CREATE TABLE core.messages (
   id BIGSERIAL PRIMARY KEY,
   chat_id INTEGER REFERENCES core.chats(id),
    role VARCHAR(20) NOT NULL,
   content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
