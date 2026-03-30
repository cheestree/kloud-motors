BEGIN;

CREATE DATABASE "chat-db";

CREATE TABLE "chat-db".chat (
    user_id INTEGER,
    listing_id INTEGER,
    make VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    chat_id uuid NOT NULL DEFAULT gen_random_uuid(),
    PRIMARY KEY (user_id, listing_id, chat_id)
);

COMMIT;