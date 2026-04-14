CREATE TABLE IF NOT EXISTS periodicity(
  task_id BIGINT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
  period_type TEXT NOT NULL CHECK (period_type IN ('daily', 'monthly', 'custom', 'even_odd')),
  days_interval INT CHECK (days_interval > 0),
  day_of_month INT CHECK (day_of_month BETWEEN 1 AND 30),
  custom_dates TIMESTAMPTZ[],
  even_odd TEXT CHECK(even_odd IN ('even', 'odd')),
  last_created_at TIMESTAMPTZ DEFAULT NOW()
);