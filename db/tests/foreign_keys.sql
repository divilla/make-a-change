-- Run with psql -v ON_ERROR_STOP=1 -f db/tests/foreign_keys.sql after init.sql.
begin;
\ir fixtures.sql

do
$$
declare
    _project_id bigint;
    _epic_id bigint;
    _change_id bigint;
    _testcase_id bigint;
    _missing_id bigint;
    _relation record;
begin
    _project_id := pg_temp.create_project('foreign key test');
    insert into public.epic (project_id, name) values (_project_id, 'epic') returning id into _epic_id;
    insert into public.change (project_id, epic_id) values (_project_id, _epic_id) returning id into _change_id;
    insert into public.testcase (change_id, scenario) values (_change_id, 'scenario') returning id into _testcase_id;

    -- Every live relationship rejects invalid inserts, reassignment, and parent deletion/key changes.
    for _relation in
        select * from (values
            ('epic', 'project_id', 'project', _epic_id, _project_id,
             'insert into public.epic (project_id, name) values ($1, ''invalid'')'),
            ('change', 'project_id', 'project', _change_id, _project_id,
             'insert into public.change (project_id) values ($1)'),
            ('change', 'epic_id', 'epic', _change_id, _epic_id,
             format('insert into public.change (project_id, epic_id) values (%s, $1)', _project_id)),
            ('testcase', 'change_id', 'change', _testcase_id, _change_id,
             'insert into public.testcase (change_id, scenario) values ($1, ''invalid'')')
        ) as relationships(child_table, child_column, parent_table, child_id, parent_id, insert_sql)
    loop
        execute format('select coalesce(max(id), 0) + 1 from public.%I', _relation.parent_table)
            into _missing_id;

        begin
            execute _relation.insert_sql using _missing_id;
            raise exception 'invalid insert accepted for %.%', _relation.child_table, _relation.child_column;
        exception when foreign_key_violation then null;
        end;

        begin
            execute format('update public.%I set %I = $1 where id = $2',
                           _relation.child_table, _relation.child_column)
                using _missing_id, _relation.child_id;
            raise exception 'invalid reassignment accepted for %.%', _relation.child_table, _relation.child_column;
        exception when foreign_key_violation then null;
        end;

        begin
            execute format('delete from public.%I where id = $1', _relation.parent_table)
                using _relation.parent_id;
            raise exception 'referenced parent deletion accepted for %.%', _relation.child_table, _relation.child_column;
        exception when foreign_key_violation then null;
        end;

        begin
            execute format('update public.%I set id = $1 where id = $2', _relation.parent_table)
                using _missing_id, _relation.parent_id;
            raise exception 'referenced parent key change accepted for %.%', _relation.child_table, _relation.child_column;
        exception when foreign_key_violation then null;
        end;
    end loop;

    -- An epic is optional, and deleting children first remains supported.
    update public.change set epic_id = null where id = _change_id;
    delete from public.epic where id = _epic_id;
    delete from public.testcase where id = _testcase_id;
    delete from public.change where id = _change_id;
    delete from public.project where id = _project_id;

end;
$$;

rollback;
