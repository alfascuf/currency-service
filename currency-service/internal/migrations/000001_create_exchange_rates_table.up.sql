-- Create exchange_rates table
CREATE TABLE IF NOT EXISTS exchange_rates (
                                              id SERIAL PRIMARY KEY,
                                              base VARCHAR(10) NOT NULL,
    target VARCHAR(10) NOT NULL,
    rate DECIMAL(20, 6) NOT NULL,
    date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(base, target, date)
    );

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_base_target_date
    ON exchange_rates(base, target, date);
