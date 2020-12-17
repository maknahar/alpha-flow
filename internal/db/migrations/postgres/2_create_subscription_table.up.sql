CREATE TABLE subscriptions
(
    user_id    int                      NOT NULL REFERENCES users (id),
    pair       text                     NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (user_id, pair)
);


CREATE TRIGGER set_updated_time
    BEFORE UPDATE
    ON subscriptions
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
