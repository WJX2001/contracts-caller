DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'uint256') THEN
        CREATE DOMAIN UINT256 AS NUMERIC
            CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
    ELSE
        ALTER DOMAIN UINT256 DROP CONSTRAINT uint256_check;
        ALTER DOMAIN UINT256 ADD
            CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
    END IF;
END $$;


CREATE TABLE IF NOT EXISTS block_headers (
    hash        VARCHAR PRIMARY KEY,
    parent_hash VARCHAR NOT NULL UNIQUE,
    number      UINT256 NOT NULL UNIQUE,
    timestamp   INTEGER NOT NULL UNIQUE CHECK (timestamp > 0),
    rlp_bytes   VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS block_headers_timestamp ON block_headers(timestamp);
CREATE INDEX IF NOT EXISTS block_headers_number ON block_headers(number);


CREATE TABLE IF NOT EXISTS contract_events (
    guid             VARCHAR PRIMARY KEY,
    block_hash       VARCHAR NOT NULL REFERENCES block_headers(hash) ON DELETE CASCADE,
    contract_address VARCHAR NOT NULL,
    transaction_hash VARCHAR NOT NULL,
    log_index        INTEGER NOT NULL,
    event_signature  VARCHAR NOT NULL,
    timestamp        INTEGER NOT NULL CHECK (timestamp > 0),
    rlp_bytes        VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS contract_events_timestamp ON contract_events(timestamp);
CREATE INDEX IF NOT EXISTS contract_events_block_hash ON contract_events(block_hash);
CREATE INDEX IF NOT EXISTS contract_events_event_signature ON contract_events(event_signature);
CREATE INDEX IF NOT EXISTS contract_events_contract_address ON contract_events(contract_address);

CREATE TABLE IF NOT EXISTS event_blocks(
    guid        VARCHAR PRIMARY KEY,
    hash        VARCHAR NOT NULL,
    parent_hash VARCHAR NOT NULL,
    number      UINT256 NOT NULL,
    timestamp   INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS event_blocks_timestamp ON event_blocks(timestamp);
CREATE INDEX IF NOT EXISTS event_blocks_number ON event_blocks(number);

CREATE TABLE IF NOT EXISTS proxy_created (
    guid                          VARCHAR PRIMARY KEY,
    proxy_address                 VARCHAR NOT NULL,
    timestamp                     INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS proxy_created_proxy_address ON proxy_created(proxy_address);

CREATE TABLE IF NOT EXISTS request_sent (
    guid                          VARCHAR PRIMARY KEY,
    request_id                    UINT256 NOT NULL,
    num_words                     UINT256 NOT NULL,
    vrf_address                   VARCHAR NOT NULL,
    status                        SMALLINT NOT NULL DEFAULT 0,
    timestamp                     INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS request_sent_request_id ON request_sent(request_id);

CREATE TABLE IF NOT EXISTS fill_random_words (
    guid                          VARCHAR PRIMARY KEY,
    request_id                    UINT256 NOT NULL,
    random_words                  VARCHAR NOT NULL,
    timestamp                     INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS fill_random_words_request_id ON request_sent(request_id);
