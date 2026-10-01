-- Run after init.sql, seed.sql, and seed-demo.sql.
begin;

do $$
begin
    assert (select count(*) = 1 from public.config), 'one default config';
    assert (select project_docs = array['brief', 'prd', 'plan', 'comment']
        and epic_docs = array['brief', 'prd', 'plan', 'comment']
        and change_docs = array['brief', 'spec', 'pr', 'review', 'comment']
        and change_phases = array['backlog', 'todo', 'in-progress', 'in-review', 'in-test', 'in-prod']
        and change_colors = array['15', '14', '10', '11', '12', '13']
        and change_types = array['feature', 'fix', 'refactor', 'upgrade', 'chore', 'docs', 'test', 'ci',
                                 'security', 'migration', 'revert', 'spike']
        from public.config where slug = 'default'), 'default configuration values';
    assert (select array_agg(name order by name) = array['demo1', 'demo2', 'demo3'] from public.project),
        'demo projects preserved';
    assert not exists (
        select from public.project p left join public.config c on c.slug = p.config_slug
        where p.config_slug <> 'default' or c.slug is null
    ), 'demo projects use the default configuration';
    assert (select count(*) = 5 from public.epic), 'five demo epics';
    assert (select count(*) = 200 from public.change), '200 demo changes';
    assert (select count(*) = 600 from public.testcase), 'three test cases per demo change';
    assert not exists (
        select c.id from public.change c left join public.testcase tc on tc.change_id = c.id
        group by c.id having count(tc.id) <> 3
    ), 'every demo change has three test cases';
    assert exists (select from public.testcase where done)
        and exists (select from public.testcase where not done), 'mixed completion states';
    assert not exists (
        select from public.vw_change_list c where c.total_tc <> 3 or c.done_tc < 1
            or c.done_tc <> (select count(*) from public.testcase t where t.change_id = c.id and t.done)
    ), 'list counts match actual test cases';
    assert not exists (
        select from public.vw_change_details d join public.vw_change_list c using (id)
        where d.total_tc <> c.total_tc or d.done_tc <> c.done_tc
    ), 'detail counts match list counts';
    assert not exists (
        select from public.vw_epic_list e where e.total_tc <> (
            select count(*) from public.testcase t join public.change c on c.id = t.change_id where c.epic_id = e.id
        ) or e.done_tc <> (
            select count(*) from public.testcase t join public.change c on c.id = t.change_id where c.epic_id = e.id and t.done
        ) or e.change_count <> (select count(*) from public.change c where c.epic_id = e.id)
    ), 'epic counts match assigned changes';
    assert (select change_count = 200 from public.vw_project_list where name = 'demo1'), 'demo1 project counts';
    assert not exists (select from public.vw_project_list where name <> 'demo1' and change_count <> 0), 'empty project counts';
    assert not exists (
        select from public.doc d left join public.change c on d.ref_id = c.id
        where d.ref_table = 'change'
            and (c.id is null or d.doc_type not in ('brief', 'spec', 'pr') or d.body = '' or d.agent_edit)
    ), 'seeded change documents belong to live changes and contain human-authored bodies';
    assert (select count(*)=3 from public.doc where ref_table='project'), 'each demo project has a brief';
    assert (select count(*)=5 from public.doc where ref_table='epic'), 'each demo epic has a brief';
    assert not exists (
        select from public.doc d
        where d.ref_table in ('project','epic')
            and (d.doc_type <> 'brief' or d.body = '' or d.agent_edit
                or (d.ref_table='project' and not exists(select from public.project p where p.id=d.ref_id))
                or (d.ref_table='epic' and not exists(select from public.epic e where e.id=d.ref_id)))
    ), 'shared briefs belong to live parents and contain human-authored bodies';
    assert not exists (
        select from public.doc d left join public.doc_active a
            on a.doc_id=d.id and a.ref_table=d.ref_table and a.ref_id=d.ref_id and a.doc_type=d.doc_type
        where a.doc_id is null
    ), 'every seeded document has a matching active reference';
    assert (select count(*) from public.doc_active) = (select count(*) from public.doc), 'no stale active references remain';
    assert not exists (
        select from public.change c where
            (select count(*) from public.doc d where d.ref_table = 'change' and d.ref_id = c.id and d.doc_type = 'brief') <> 1
            or (select count(*) from public.doc d where d.ref_table = 'change' and d.ref_id = c.id and d.doc_type = 'pr') <> 1
            or (select count(*) from public.doc d where d.ref_table = 'change' and d.ref_id = c.id and d.doc_type = 'spec')
                <> case when c.ref % 11 = 0 then 0 else 1 end
    ), 'each change has its brief, PR, and optional spec documents';
    assert not exists (
        select from public.change c where c.ref is null or c.slug is null or c.pr_url = ''
    ), 'fixture identifiers and PR links are populated';
    assert (select count(distinct ref) = 200 and min(ref) = 201 and max(ref) = 400 from public.change),
        'fixture references remain unique and stable';
    assert (select last_ref = 400 from public.project where name = 'demo1'), 'reference counter tracks fixtures';
    assert not exists (
        select from public.change c join public.project p on p.id = c.project_id
        join public.config cfg on cfg.slug = p.config_slug
        where not (c.change_phase = any(cfg.change_phases)) or not (c.change_types <@ cfg.change_types)
    ), 'demo phases and types come from project configuration';
    assert (select count(*) = 80 from public.change where change_phase = 'backlog'), '40 percent backlog';
    assert not exists (
        select phase from public.project p join public.config cfg on cfg.slug = p.config_slug
        cross join lateral unnest(cfg.change_phases) as phases(phase)
        where p.name = 'demo1' and phase <> 'backlog'
            and (select count(*) from public.change c where c.project_id = p.id and c.change_phase = phase) <> 24
    ), 'remaining changes spread evenly across configured phases';
    assert (select count(*) = 60 from public.change where epic_id is null), '30 percent standalone changes';
    assert exists (select from public.change where cardinality(change_types) = 0), 'optional empty change types';
end;
$$;
rollback;
