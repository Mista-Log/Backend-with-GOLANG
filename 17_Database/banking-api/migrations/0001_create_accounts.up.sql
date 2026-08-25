CREATE TABLE accounts (
	id      INTEGER PRIMARY KEY AUTOINCREMENT,
	owner   TEXT NOT NULL UNIQUE,
	balance REAL NOT NULL CHECK (balance >= 0)
);

-- A separate audit log of every transfer — written INSIDE the same
-- transaction as the two balance updates, so the log entry and the
-- balance changes are always consistent with each other (all three writes
-- commit together, or none of them do).
CREATE TABLE transfers (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	from_id     INTEGER NOT NULL,
	to_id       INTEGER NOT NULL,
	amount      REAL NOT NULL CHECK (amount > 0),
	created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (from_id) REFERENCES accounts(id),
	FOREIGN KEY (to_id) REFERENCES accounts(id)
);

-- Indexes on the foreign key columns — without these, "show me every
-- transfer involving account X" would need a full table scan of transfers.
CREATE INDEX idx_transfers_from ON transfers(from_id);
CREATE INDEX idx_transfers_to ON transfers(to_id);
