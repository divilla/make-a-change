begin;

truncate table public.config;

insert into public.config (
    project_docs,
    epic_docs,
    change_docs,
    change_phases,
    change_colors,
    change_types
) values (
    array['brief', 'prd'],
    array['brief', 'prd'],
    array['brief', 'spec', 'pr', 'plan', 'review', 'comment'],
    array['backlog', 'todo', 'in-progress', 'in-review', 'in-test', 'in-prod'],
    array['15', '14', '10', '11', '12', '13'],
    array['feature', 'fix', 'refactor', 'upgrade', 'chore', 'docs', 'test', 'ci', 'security', 'migration', 'revert', 'spike']
);

commit;
