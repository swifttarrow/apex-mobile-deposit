CREATE TABLE IF NOT EXISTS transfers (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL,
    amount REAL NOT NULL,
    state TEXT NOT NULL,
    vendor_response TEXT,
    front_image_path TEXT,
    back_image_path TEXT,
    micr_data TEXT,
    ocr_amount REAL,
    entered_amount REAL,
    transaction_id TEXT,
    contribution_type TEXT,
    settlement_batch_id TEXT,
    settlement_ack_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id TEXT PRIMARY KEY,
    transfer_id TEXT NOT NULL,
    to_account_id TEXT NOT NULL,
    from_account_id TEXT NOT NULL,
    type TEXT NOT NULL,
    memo TEXT NOT NULL,
    sub_type TEXT NOT NULL,
    transfer_type TEXT NOT NULL,
    currency TEXT NOT NULL,
    amount REAL NOT NULL,
    source_application_id TEXT NOT NULL,
    contribution_type TEXT,
    created_at TEXT NOT NULL,
    is_reversal INTEGER NOT NULL DEFAULT 0,
    reversal_fee REAL NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS operator_actions (
    id TEXT PRIMARY KEY,
    transfer_id TEXT NOT NULL,
    action TEXT NOT NULL,
    operator_id TEXT NOT NULL,
    note TEXT,
    contribution_type_override TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key TEXT PRIMARY KEY,
    response_body TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS check_images (
    id TEXT PRIMARY KEY,
    transfer_id TEXT NOT NULL,
    image_type TEXT NOT NULL,
    image_data TEXT NOT NULL,
    created_at TEXT NOT NULL
);
