-- Run with psql -v ON_ERROR_STOP=1 -f db/tests/test_case.sql after init.sql.
begin;

do
$$
declare
    _project_id bigint;
    _source_epic_id bigint;
    _source_id bigint;
    _test_case_id bigint;
begin
    insert into public.project (name) values ('testcase procedure test') returning id into _project_id;
    insert into public.epic (project_id, name) values (_project_id, 'source') returning id into _source_epic_id;
    insert into public.change (project_id, epic_id, title)
    values (_project_id, _source_epic_id, 'source') returning id into _source_id;
    insert into public.test_case (change_id, scenario)
    values (_source_id, 'test scenario') returning id into _test_case_id;

    -- Newly created cases are incomplete, but must still be counted.
    call public.sp_test_case_update_done(_test_case_id, false);
    assert (select done_tc = 0 and total_tc = 1 from public.change where id = _source_id), 'create change counts';
    assert (select done_tc = 0 and total_tc = 1 from public.epic where id = _source_epic_id), 'create epic counts';

    call public.sp_test_case_update_done(_test_case_id, true);
    assert (select done and version = 0 from public.test_case where id = _test_case_id), 'done without version change';
    assert (select done_tc = 1 and completed = 100 from public.change where id = _source_id), 'done change counts';
    assert (select done_tc = 1 and completed = 100 from public.epic where id = _source_epic_id), 'done epic counts';

    call public.sp_test_case_update_done(_test_case_id, true);
    assert (select done_tc = 1 and total_tc = 1 from public.change where id = _source_id), 'repeated done update';

    call public.sp_test_case_update_done(_test_case_id, false);
    assert (select not done from public.test_case where id = _test_case_id), 'clear done';
    assert (select done_tc = 0 and total_tc = 1 from public.change where id = _source_id), 'clear change counts';
    assert (select done_tc = 0 and total_tc = 1 from public.epic where id = _source_epic_id), 'clear epic counts';

    call public.sp_test_case_update_done(_test_case_id, true);
    call public.sp_test_case_delete(_test_case_id);
    assert not exists (select from public.test_case where id = _test_case_id), 'deleted testcase';
    assert exists (select from public.test_case_history where id = _test_case_id and deleted), 'deletion history';
    assert (select done_tc = 0 and total_tc = 0 and completed = 0 from public.change where id = _source_id), 'delete last change counts';
    assert (select done_tc = 0 and total_tc = 0 from public.epic where id = _source_epic_id), 'delete last epic counts';
end;
$$;

rollback;
