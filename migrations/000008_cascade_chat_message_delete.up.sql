ALTER TABLE core.messages
    DROP CONSTRAINT messages_chat_id_fkey,
    ADD CONSTRAINT messages_chat_id_fkey
        FOREIGN KEY (chat_id)
        REFERENCES core.chats(id)
        ON DELETE CASCADE;
