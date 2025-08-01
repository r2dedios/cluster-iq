-- ## Tables definitions ##
-- Cloud Providers
CREATE TABLE IF NOT EXISTS providers (
  name TEXT PRIMARY KEY
);

-- Default values for Cloud Providers table
INSERT INTO
  providers(name)
VALUES
  ('AWS'),
  ('GCP'),
  ('Azure'),
  ('UNKNOWN')
;


-- Action Operations
CREATE TABLE IF NOT EXISTS action_operations (
  name TEXT PRIMARY KEY
);

-- Default values for Cloud Providers table
INSERT INTO
  action_operations(name)
VALUES
  ('PowerOnCluster'),
  ('PowerOffCluster')
;

-- Status
CREATE TABLE IF NOT EXISTS status (
  value TEXT PRIMARY KEY
);

-- Default values for Status table
INSERT INTO
  status(value)
VALUES
  ('Running'),
  ('Stopped'),
  ('Terminated')
;


-- Accounts
CREATE TABLE IF NOT EXISTS accounts (
  id TEXT,
  name TEXT PRIMARY KEY,
  provider TEXT REFERENCES providers(name),
  cluster_count INTEGER,
  last_scan_timestamp TIMESTAMP WITH TIME ZONE,
  total_cost NUMERIC(12,2) DEFAULT 0.0,
  last_15_days_cost NUMERIC(12,2) DEFAULT 0.0,
  last_month_cost NUMERIC(12,2) DEFAULT 0.0,
  current_month_so_far_cost NUMERIC(12,2) DEFAULT 0.0
);


-- Clusters
CREATE TABLE IF NOT EXISTS clusters (
  -- id is the result of joining: "name+infra_id+account"
  id TEXT PRIMARY KEY,
  name TEXT,
  infra_id TEXT,
  provider TEXT REFERENCES providers(name),
  status TEXT REFERENCES status(value),
  region TEXT,
  account_name TEXT REFERENCES accounts(name) ON DELETE CASCADE,
  console_link TEXT,
  instance_count INTEGER,
  last_scan_timestamp TIMESTAMP WITH TIME ZONE,
  creation_timestamp TIMESTAMP WITH TIME ZONE,
  age INT,
  owner TEXT,
  total_cost NUMERIC(12,2) DEFAULT 0.0,
  last_15_days_cost NUMERIC(12,2) DEFAULT 0.0,
  last_month_cost NUMERIC(12,2) DEFAULT 0.0,
  current_month_so_far_cost NUMERIC(12,2) DEFAULT 0.0
);


-- Instances
CREATE TABLE IF NOT EXISTS instances (
  id TEXT PRIMARY KEY,
  name TEXT,
  provider TEXT REFERENCES providers(name),
  instance_type TEXT,
  availability_zone TEXT,
  status TEXT REFERENCES status(value),
  cluster_id TEXT REFERENCES clusters(id) ON DELETE CASCADE,
  last_scan_timestamp TIMESTAMP WITH TIME ZONE,
  creation_timestamp TIMESTAMP WITH TIME ZONE,
  age INT,
  daily_cost NUMERIC(12,2) DEFAULT 0.0,
  total_cost NUMERIC(12,2) DEFAULT 0.0
);


-- Instances Tags
CREATE TABLE IF NOT EXISTS tags (
  key TEXT,
  value TEXT,
  instance_id TEXT REFERENCES instances(id) ON DELETE CASCADE,
  PRIMARY KEY (key, instance_id)
);


-- Instances expenses
CREATE TABLE IF NOT EXISTS expenses (
  instance_id TEXT REFERENCES instances(id) ON DELETE CASCADE,
  date DATE,
  amount NUMERIC(12,2) DEFAULT 0.0,
  PRIMARY KEY (instance_id, date)
);

-- Action types table
CREATE TABLE IF NOT EXISTS action_types (
  name TEXT PRIMARY KEY
);

-- Default values for Action Types
INSERT INTO
  action_types(name)
VALUES
  ('cron_action'),
  ('scheduled_action')
;

-- Action Status table
CREATE TABLE IF NOT EXISTS action_status (
  name TEXT PRIMARY KEY
);

-- Default values for Action Types
INSERT INTO
  action_status(name)
VALUES
  ('Success'),
  ('Failed'),
  ('Pending'),
  ('Unknown')
;

-- Scheduled actions
CREATE TABLE IF NOT EXISTS schedule (
  id BIGINT GENERATED ALWAYS AS IDENTITY NOT NULL,
  type TEXT REFERENCES action_types(name),
  time TIMESTAMP WITH TIME ZONE,
  cron_exp TEXT,
  operation TEXT REFERENCES action_operations(name),
  target TEXT REFERENCES clusters(id) ON DELETE CASCADE,
  status TEXT REFERENCES action_status(name),
  enabled BOOLEAN
);


-- Audit logs
CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGINT GENERATED ALWAYS AS IDENTITY NOT NULL,
  event_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  triggered_by TEXT NOT NULL,
  action_name TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  result TEXT NOT NULL,
  description TEXT NULL,
  severity TEXT DEFAULT 'info'::TEXT NOT NULL,
  CONSTRAINT audit_logs_pkey PRIMARY KEY (id),
  CONSTRAINT audit_logs_resource_type_check CHECK ((resource_type = ANY (ARRAY['cluster'::TEXT, 'instance'::TEXT])))
);

-- ## Functions ##
-- Updates the total cost of an instance after a new expense record is inserted
CREATE OR REPLACE FUNCTION update_instance_total_costs_after_insert()
  RETURNS TRIGGER
  LANGUAGE PLPGSQL
  AS
$$
BEGIN
  UPDATE instances
  SET
    total_cost = (
      SELECT SUM(amount)
      FROM expenses
      WHERE instance_id = NEW.instance_id
    )
    WHERE id = NEW.instance_id;
  RETURN NEW;
END;
$$;

-- Updates the total cost of an instance after an expense record is deleted
CREATE OR REPLACE FUNCTION update_instance_total_costs_after_delete()
  RETURNS TRIGGER
  LANGUAGE PLPGSQL
  AS
$$
BEGIN
  UPDATE instances
  SET
    total_cost = (
      SELECT SUM(amount)
      FROM expenses
      WHERE instance_id = OLD.instance_id
    )
    WHERE id = OLD.instance_id;
  RETURN OLD;
END;
$$;

-- Updates the daily cost of an instance after a new expense record is inserted
CREATE OR REPLACE FUNCTION update_instance_daily_costs_after_insert()
  RETURNS TRIGGER
  LANGUAGE PLPGSQL
  AS
$$
BEGIN
  UPDATE instances
  SET
    daily_cost = (
      SELECT COALESCE(SUM(amount)/NULLIF(COUNT(*), 0), 0)
      FROM expenses
      WHERE instance_id = NEW.instance_id
    )
    WHERE id = NEW.instance_id;
  RETURN NEW;
END;
$$;

-- Updates the daily cost of an instance after an expense record is deleted
CREATE OR REPLACE FUNCTION update_instance_daily_costs_after_delete()
  RETURNS TRIGGER
  LANGUAGE PLPGSQL
  AS
$$
BEGIN
  UPDATE instances
  SET
    daily_cost = (
      SELECT COALESCE(SUM(amount)/NULLIF(COUNT(*), 0), 0)
      FROM expenses
      WHERE instance_id = NEW.instance_id
    )
    WHERE id = OLD.instance_id;
  RETURN OLD;
END;
$$;

-- Updates the total cost of a cluster based on its associated instances
CREATE OR REPLACE FUNCTION update_cluster_cost_info()
  RETURNS TRIGGER
  LANGUAGE PLPGSQL
  AS
$$
BEGIN
  UPDATE clusters
  SET
    total_cost = (
      SELECT COALESCE(SUM(total_cost), 0) as sum
      FROM instances
      WHERE cluster_id = NEW.cluster_id
    ),
    last_15_days_cost = (
      SELECT COALESCE(SUM(expenses.amount), 0)
      FROM instances
      JOIN expenses ON instances.id = expenses.instance_id
      WHERE instances.cluster_id = NEW.cluster_id
        AND expenses.date >= NOW()::date - interval '15 day'
    ),
    last_month_cost = (
      SELECT COALESCE(SUM(expenses.amount), 0)
      FROM instances
      JOIN expenses ON instances.id = expenses.instance_id
      WHERE instances.cluster_id = NEW.cluster_id
        AND EXTRACT(YEAR FROM NOW()::date - interval '1 month') = EXTRACT(YEAR FROM expenses.date)
        AND EXTRACT(MONTH FROM NOW()::date - interval '1 month') = EXTRACT(MONTH FROM expenses.date)
    ),
    current_month_so_far_cost = (
      SELECT COALESCE(SUM(expenses.amount), 0)
      FROM instances
      JOIN expenses ON instances.id = expenses.instance_id
      WHERE instances.cluster_id = NEW.cluster_id
        AND (EXTRACT(MONTH FROM NOW()::date) = EXTRACT(MONTH FROM expenses.date)
      )
    )
    WHERE id = NEW.cluster_id;
  RETURN NEW;
END;
$$;

-- Updates the total cost of an account based on its associated clusters
CREATE OR REPLACE FUNCTION update_account_cost_info()
  RETURNS TRIGGER
  LANGUAGE PLPGSQL
  AS
$$
BEGIN
  UPDATE accounts
  SET
    total_cost = (
      SELECT COALESCE(SUM(clusters.total_cost), 0)
      FROM clusters
      WHERE account_name = NEW.account_name
    ),
    last_15_days_cost = (
      SELECT COALESCE(SUM(clusters.last_15_days_cost), 0)
      FROM clusters
      WHERE account_name = NEW.account_name
    ),
    last_month_cost = (
      SELECT COALESCE(SUM(clusters.last_month_cost), 0)
      FROM clusters
      WHERE account_name = NEW.account_name
    ),
    current_month_so_far_cost = (
      SELECT COALESCE(SUM(clusters.current_month_so_far_cost), 0)
      FROM clusters
      WHERE account_name = NEW.account_name
    )
    WHERE name = NEW.account_name;
  RETURN NEW;
END;
$$;

-- ## Maintenance Functions ##
-- Marks instances as 'Terminated' if they haven't been scanned in the last 24 hours
CREATE OR REPLACE FUNCTION check_terminated_instances()
RETURNS void AS $$
BEGIN
  UPDATE instances
  SET status = 'Terminated'
  WHERE last_scan_timestamp < NOW() - INTERVAL '1 day'
    AND status IS DISTINCT FROM 'Terminated';
END;
$$ LANGUAGE plpgsql;

-- Marks clusters as 'Terminated' if they haven't been scanned in the last 24 hours
CREATE OR REPLACE FUNCTION check_terminated_clusters()
RETURNS void AS $$
BEGIN
  UPDATE clusters
  SET status = 'Terminated'
  WHERE last_scan_timestamp < NOW() - INTERVAL '1 day'
    AND status IS DISTINCT FROM 'Terminated';
END;
$$ LANGUAGE plpgsql;

-- ## Triggers ##
-- Trigger to update instance total cost after an expense is inserted
CREATE TRIGGER update_instance_total_cost_after_insert
AFTER INSERT
ON expenses
FOR EACH ROW
  EXECUTE PROCEDURE update_instance_total_costs_after_insert();

-- Trigger to update instance total cost after an expense is updated
CREATE TRIGGER update_instance_total_cost_after_update
AFTER UPDATE
ON expenses
FOR EACH ROW
  EXECUTE PROCEDURE update_instance_total_costs_after_insert();

-- Trigger to update instance total cost after an expense is deleted
CREATE TRIGGER update_instance_total_cost_after_delete
AFTER DELETE
ON expenses
FOR EACH ROW
  EXECUTE PROCEDURE update_instance_total_costs_after_delete();

-- Trigger to update instance daily cost after an expense is inserted
CREATE TRIGGER update_instance_daily_cost_after_insert
AFTER INSERT
ON expenses
FOR EACH ROW
  EXECUTE PROCEDURE update_instance_daily_costs_after_insert();

-- Trigger to update instance daily cost after an expense is updated
CREATE TRIGGER update_instance_daily_cost_after_update
AFTER UPDATE
ON expenses
FOR EACH ROW
  EXECUTE PROCEDURE update_instance_daily_costs_after_insert();

-- Trigger to update instance daily cost after an expense is deleted
CREATE TRIGGER update_instance_daily_cost_after_delete
AFTER DELETE
ON expenses
FOR EACH ROW
  EXECUTE PROCEDURE update_instance_daily_costs_after_delete();

-- Trigger to update cluster costs info
CREATE TRIGGER update_cluster_cost_info
AFTER UPDATE
ON instances
FOR EACH ROW
  EXECUTE PROCEDURE update_cluster_cost_info();

-- Trigger to update account total cost after a cluster is updated
CREATE TRIGGER update_account_cost_info
AFTER UPDATE
ON clusters
FOR EACH ROW
  EXECUTE PROCEDURE update_account_cost_info();
