-- Session-local fixtures; no application schema changes or seed dependency.
create or replace function pg_temp.create_project(_name text) returns bigint
language sql as $$
    insert into public.project (name) values (_name) returning id;
$$;
