-- Run with psql -v ON_ERROR_STOP=1 -f db/tests/change_epic.sql after init.sql.
begin;

do
$$
declare
    _project_id bigint;
    _other_project_id bigint;
    _epic_id bigint;
    _another_epic_id bigint;
    _other_epic_id bigint;
    _change_id bigint;
    _unassigned_id bigint;
begin
    insert into public.project (name) values ('epic owner') returning id into _project_id;
    insert into public.project (name) values ('other owner') returning id into _other_project_id;
    insert into public.epic (project_id, name) values (_project_id, 'first') returning id into _epic_id;
    insert into public.epic (project_id, name) values (_project_id, 'second') returning id into _another_epic_id;
    insert into public.epic (project_id, name) values (_other_project_id, 'other') returning id into _other_epic_id;
    insert into public.change (project_id, epic_id) values (_project_id, _epic_id) returning id into _change_id;

    begin
        insert into public.change (project_id, epic_id) values (_project_id, _other_epic_id);
        raise exception 'change inserted with another project''s epic';
    exception when foreign_key_violation then null;
    end;

    begin
        update public.change set epic_id = _other_epic_id where id = _change_id;
        raise exception 'change assigned to another project''s epic';
    exception when foreign_key_violation then null;
    end;

    begin
        update public.change set project_id = _other_project_id where id = _change_id;
        raise exception 'change moved without updating its epic';
    exception when foreign_key_violation then null;
    end;

    begin
        update public.epic set project_id = _other_project_id where id = _epic_id;
        raise exception 'referenced epic moved to another project';
    exception when foreign_key_violation then null;
    end;

    -- Another epic in the same project remains a valid assignment.
    update public.change set epic_id = _another_epic_id where id = _change_id;
    assert (select epic_id = _another_epic_id from public.change where id = _change_id),
        'same-project reassignment failed';

    -- Both columns can be changed together to another valid pair.
    update public.change set project_id = _other_project_id, epic_id = _other_epic_id where id = _change_id;
    assert (select project_id = _other_project_id and epic_id = _other_epic_id
            from public.change where id = _change_id), 'valid project/epic pair rejected';

    -- An epic is optional on both insert and update.
    insert into public.change (project_id) values (_project_id) returning id into _unassigned_id;
    update public.change set epic_id = null where id = _change_id;
    assert (select epic_id is null from public.change where id = _unassigned_id), 'unassigned insert failed';
    assert (select epic_id is null from public.change where id = _change_id), 'clearing epic failed';
end;
$$;

rollback;
