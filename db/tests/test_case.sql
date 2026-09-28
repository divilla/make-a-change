-- Run with psql -v ON_ERROR_STOP=1 -f db/tests/test_case.sql after init.sql.
begin;
\ir fixtures.sql

do $$
declare
    _project_id bigint := pg_temp.create_project('testcase test');
    _epic_id bigint;
    _change_id bigint;
    _other_change_id bigint;
    _testcase_id bigint;
begin
    insert into public.epic (project_id, name) values (_project_id, 'source') returning id into _epic_id;
    insert into public.change (project_id, epic_id) values (_project_id, _epic_id) returning id into _change_id;
    insert into public.change (project_id) values (_project_id) returning id into _other_change_id;
    assert (select total_tc = 0 and done_tc = 0 from public.vw_change_details where id = _change_id), 'empty counts';
    insert into public.testcase (change_id, scenario) values (_change_id, 'original') returning id into _testcase_id;
    assert (select not done and created_at = updated_at from public.testcase where id = _testcase_id), 'creation defaults';
    assert (select done_tc = 0 and total_tc = 1 from public.vw_change_list where id = _change_id), 'creation counts';

    update public.testcase set done = true where id = _testcase_id;
    assert (select done_tc = 1 and total_tc = 1 from public.vw_change_details where id = _change_id), 'done counts';
    assert (select done_tc = 1 and total_tc = 1 from public.vw_epic where id = _epic_id), 'epic done counts';
    update public.testcase set scenario = 'edited', updated_at = now() where id = _testcase_id;
    assert (select scenario = 'edited' and done from public.testcase where id = _testcase_id), 'scenario preserves done';
    update public.testcase set done = true where id = _testcase_id;
    assert (select done_tc = 1 from public.vw_change_list where id = _change_id), 'repeated done counts';
    update public.testcase set done = false where id = _testcase_id;
    assert (select done_tc = 0 and total_tc = 1 from public.vw_epic where id = _epic_id), 'cleared done counts';

    update public.testcase set change_id = _other_change_id where id = _testcase_id;
    assert (select total_tc = 0 from public.vw_change_list where id = _change_id), 'source excludes moved testcase';
    assert (select total_tc = 0 from public.vw_epic where id = _epic_id), 'epic excludes moved testcase';
    assert (select total_tc = 1 from public.vw_change_details where id = _other_change_id), 'standalone destination counts';
    delete from public.testcase where id = _testcase_id;
    assert (select done_tc = 0 and total_tc = 0 from public.vw_change_details where id = _other_change_id), 'delete last counts';
end;
$$;
rollback;
