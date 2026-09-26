begin;

truncate table public.change_phase;
truncate table public.change_type;
truncate table public.config;

insert into public.change_phase (slug, priority)
values
    ('backlog', 0),
    ('progress', 1),
    ('review', 2),
    ('staging', 3),
    ('production', 4),
    ('rejected', 5)
on conflict (slug) do update
set priority = excluded.priority;

insert into public.change_type (slug, priority)
values
    ('feature', 0),
    ('fix', 1),
    ('refactor', 2),
    ('upgrade', 3),
    ('chore', 4),
    ('docs', 5),
    ('test', 6),
    ('ci', 7),
    ('security', 8),
    ('migration', 9),
    ('revert', 10),
    ('spike', 11)
on conflict (slug) do update
set priority = excluded.priority;

insert into public.config (
    slug,
    change_phases,
    change_types,
    change_docs
) values (
    'default',
    array(select slug from public.change_phase order by priority, slug),
    array(select slug from public.change_type order by priority, slug),
    array['brief', 'spec', 'pr']
);

commit;
