CREATE TABLE IF NOT EXISTS otc_negotiations (
    id               BIGSERIAL PRIMARY KEY,
    ticker           VARCHAR(20)    NOT NULL,
    seller_id        BIGINT         NOT NULL,
    seller_type      VARCHAR(10)    NOT NULL DEFAULT 'CLIENT',
    buyer_id         BIGINT         NOT NULL,
    buyer_type       VARCHAR(10)    NOT NULL DEFAULT 'CLIENT',
    amount           INT            NOT NULL,
    price_per_stock  DECIMAL(18,4)  NOT NULL,
    settlement_date  DATE           NOT NULL,
    premium          DECIMAL(18,4)  NOT NULL,
    currency         VARCHAR(10)    NOT NULL,
    last_modified    TIMESTAMP      NOT NULL DEFAULT NOW(),
    modified_by_id   BIGINT,
    modified_by_type VARCHAR(10),
    status           VARCHAR(20)    NOT NULL DEFAULT 'PENDING_SELLER'
);
