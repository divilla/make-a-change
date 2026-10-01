-- Run with psql -v ON_ERROR_STOP=1 -f db/tests/change_epic.sql after init.sql and seed.sql.
begin;
\ir fixtures.sql

do $$
declare
    _project_id bigint := pg_temp.create_project('epic owner');
    _other_project_id bigint := pg_temp.create_project('other owner');
    _epic_id bigint;
    _another_epic_id bigint;
    _foreign_epic_id bigint;
    _change_id bigint;
    _cross_project_change_id bigint;
begin
    insert into public.epic (project_id, name) values (_project_id, 'first') returning id into _epic_id;
    insert into public.epic (project_id, name) values (_project_id, 'second') returning id into _another_epic_id;
    insert into public.epic (project_id, name) values (_other_project_id, 'foreign') returning id into _foreign_epic_id;
    insert into public.change (project_id) values (_project_id) returning id into _change_id;
    insert into public.testcase (change_id, scenario, done) values (_change_id, 'scenario', true);

    assert (select epic_id is null and total_tc = 1 from public.vw_change_list where id = _change_id),
        'standalone change retains testcase counts';
    call public.sp_change_epic_update(_change_id, _epic_id);
    assert (select epic_name = 'first' from public.vw_change_details where id = _change_id),
        'assigned epic name is visible';
    assert (select change_count = 1 and done_tc = 1 and total_tc = 1 from public.vw_epic_list where id = _epic_id),
        'assigned epic counts include the change';

    -- init.sql intentionally omits the composite project/epic foreign key.
    call public.sp_change_epic_update(_change_id, _foreign_epic_id);
    assert (select epic_id = _foreign_epic_id and project_id = _project_id
            and epic_name = 'foreign' from public.vw_change_details where id = _change_id),
        'cross-project assignment preserves the change project and exposes the epic';
    assert (select change_count = 1 and done_tc = 1 and total_tc = 1
            from public.vw_epic_list where id = _foreign_epic_id),
        'cross-project epic counts include assigned changes';
    assert (select change_count = 0 and total_tc = 0 from public.vw_epic_list where id = _epic_id),
        'previous epic excludes the cross-project assignment';

    insert into public.change (project_id, epic_id) values (_project_id, _foreign_epic_id)
        returning id into _cross_project_change_id;
    assert (select change_count = 2 from public.vw_epic_list where id = _foreign_epic_id),
        'direct cross-project inserts are counted';
    delete from public.change where id = _cross_project_change_id;

    call public.sp_change_epic_update(_change_id, _epic_id);
    update public.change set project_id = _other_project_id where id = _change_id;
    assert (select project_id = _other_project_id and epic_id = _epic_id
            from public.change where id = _change_id),
        'change project reassignment preserves the epic';
    update public.change set project_id = _project_id where id = _change_id;
    update public.epic set project_id = _other_project_id where id = _epic_id;
    assert (select project_id = _other_project_id and change_count = 1 and done_tc = 1
            from public.vw_epic_list where id = _epic_id),
        'epic project reassignment preserves assigned changes and counts';

    call public.sp_change_epic_update(_change_id, _another_epic_id);
    assert (select change_count = 0 and total_tc = 0 from public.vw_epic_list where id = _epic_id),
        'previous epic counts exclude the change';
    assert (select change_count = 1 and done_tc = 1 from public.vw_epic_list where id = _another_epic_id),
        'new epic counts include the change';

    begin
        call public.sp_change_epic_update(_change_id, -1);
        raise exception 'missing epic accepted';
    exception when foreign_key_violation then null;
    end;

    call public.sp_change_epic_update(_change_id, null);
    assert (select epic_id is null and epic_name is null and total_tc = 1
            from public.vw_change_details where id = _change_id), 'clearing epic preserves the change';
    assert (select change_count = 0 and total_tc = 0 from public.vw_epic_list where id = _another_epic_id),
        'cleared epic counts exclude the change';
    call public.sp_change_epic_update(-1, _epic_id);
end;
$$;
rollback;
