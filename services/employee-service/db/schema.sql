CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE employees (
    id             BIGSERIAL PRIMARY KEY,
    first_name     VARCHAR,
    last_name      VARCHAR,
    date_of_birth  DATE,
    gender         VARCHAR,
    email          VARCHAR UNIQUE,
    phone_number   VARCHAR,
    address        VARCHAR,
    username       VARCHAR UNIQUE,
    password       VARCHAR,
    position       VARCHAR,
    department     VARCHAR,
    active         BOOLEAN,
    permissions    TEXT[],
    jmbg           VARCHAR(13) NOT NULL UNIQUE
);

CREATE INDEX IF NOT EXISTS idx_employees_first_name ON employees (first_name);
CREATE INDEX IF NOT EXISTS idx_employees_last_name  ON employees (last_name);
CREATE INDEX IF NOT EXISTS idx_employees_position   ON employees (position);

INSERT INTO employees (first_name, last_name, date_of_birth, gender, email, phone_number, address, username, password, position, department, active, permissions, jmbg)
SELECT 'Admin', 'Admin', '1990-01-01', 'M', 'admin@exbanka.com', '', '', 'admin', crypt('admin', gen_salt('bf', 10)), 'Administrator', 'IT', true, ARRAY['ADMIN', 'READ', 'WRITE', 'DELETE'], '0000000000001'
WHERE NOT EXISTS (SELECT 1 FROM employees WHERE username = 'admin');
