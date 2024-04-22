DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_database WHERE datname = 'wbstore') THEN
        CREATE DATABASE wbstore;
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_user WHERE usename = 'felix') THEN
        CREATE USER felix WITH ENCRYPTED PASSWORD '12345678';
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'felix') THEN
        CREATE ROLE felix WITH LOGIN;
    END IF;
END
$$;

GRANT CREATE ON DATABASE wbstore TO felix;

GRANT ALL PRIVILEGES ON DATABASE wbstore TO felix;