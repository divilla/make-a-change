-- Run after init.sql, seed.sql, and seed-demo.sql.
begin;

do
$$
begin
    assert (select count(*) = 1 from public.config), 'one default config';
    assert (
        select change_phases = array(select slug from public.change_phase order by priority, slug)
            and change_types = array(select slug from public.change_type order by priority, slug)
            and change_docs = array['brief', 'spec', 'pr']
        from public.config where slug = 'default'
    ), 'config matches lookup tables and supported documents';

    assert (select array_agg(name order by name) = array['demo1', 'demo2', 'demo3'] from public.project),
        'demo projects preserved';
    assert (select count(*) = 5 from public.epic), 'five demo epics';
    assert (select count(*) = 200 from public.change), '200 demo changes';
    assert (select count(*) = 600 from public.test_case), 'three test cases per demo change';
    assert not exists (
        select c.id from public.change c
        left join public.test_case tc on tc.change_id = c.id
        group by c.id having count(tc.id) <> 3
    ), 'every demo change has three test cases';

    assert (select count(*) = 600 from public.test_case_history), 'one initial history per test case';
    assert not exists (
        select from public.test_case tc
        left join public.test_case_history h on h.id = tc.id and h.version = 0
        where h.id is null or h.deleted or h.scenario <> tc.scenario
            or h.change_id <> tc.change_id or h.modified <> tc.created or tc.version <> 0
    ), 'initial test case history matches creation; completion does not increment version';
    assert exists (select from public.test_case where done)
        and exists (select from public.test_case where not done), 'mixed completion states preserved';

    assert not exists (
        select c.id from public.change c
        left join public.test_case tc on tc.change_id = c.id
        group by c.id
        having c.total_tc <> count(tc.id)
            or c.done_tc <> count(tc.id) filter (where tc.done)
    ), 'change counters match actual test cases';
    assert not exists (
        select e.id from public.epic e
        left join public.change c on c.epic_id = e.id
        group by e.id
        having e.total_tc <> coalesce(sum(c.total_tc), 0)
            or e.done_tc <> coalesce(sum(c.done_tc), 0)
    ), 'epic counters match assigned changes';

    assert not exists (
        select from public.change c
        left join public.change_history h on h.id = c.id and h.version = 0
        where h.id is null or h.doc_type <> 'brief' or h.body <> c.brief or h.deleted
    ), 'initial change history is preserved';
    assert not exists (
        select c.id from public.change c
        join public.change_history h on h.id = c.id
        group by c.id
        having max(h.version) <> c.version or count(*) <> c.version + 1
    ), 'document history versions remain contiguous';
    assert not exists (
        select from public.change c
        cross join lateral (values ('spec', c.spec), ('pr', c.pr)) as doc(kind, body)
        where doc.body <> '' and not exists (
            select from public.change_history h
            where h.id = c.id and h.doc_type = doc.kind and h.body = doc.body and not h.deleted
        )
    ), 'submitted documents have history';

    assert not exists (
        select from public.change c
        left join public.change_phase phase on phase.slug = c.change_phase
        where phase.slug is null
    ), 'all demo phases exist in the lookup table';
    assert (select count(*) = 80 from public.change where change_phase = 'backlog'), '40 percent backlog';
    assert not exists (
        select phase.slug from public.change_phase phase
        left join public.change c on c.change_phase = phase.slug
        where phase.slug <> 'backlog'
        group by phase.slug having count(c.id) <> 24
    ), 'remaining changes spread evenly across seeded phases';
    assert (select count(*) = 60 from public.change where epic_id is null), '30 percent standalone changes';
    assert exists (select from public.change where spec = '')
        and exists (select from public.change where cardinality(change_types) = 0), 'optional default states preserved';
    assert not exists (
        select from public.change c cross join lateral unnest(c.change_types) as chosen(slug)
        left join public.change_type t on t.slug = chosen.slug where t.slug is null
    ), 'all demo change types exist in the lookup table';
end;
$$;

rollback;
