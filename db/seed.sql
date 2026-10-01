begin;

insert into public.config (
    project_docs,
    epic_docs,
    change_docs,
    change_phases,
    change_colors,
    change_types
) values (
    array['brief', 'prd', 'plan', 'comment'],
    array['brief', 'prd', 'plan', 'comment'],
    array['brief', 'spec', 'pr', 'review', 'comment'],
    array['backlog', 'todo', 'in-progress', 'in-review', 'in-test', 'in-prod'],
    array['15', '14', '10', '11', '12', '13'],
    array['feature', 'fix', 'refactor', 'upgrade', 'chore', 'docs', 'test', 'ci', 'security', 'migration', 'revert', 'spike']
) on conflict (slug) do update set
     project_docs=array['brief', 'prd', 'plan', 'comment'],
     epic_docs=array['brief', 'prd', 'plan', 'comment'],
     change_docs=array['brief', 'spec', 'pr', 'review', 'comment'],
     change_phases=array['backlog', 'todo', 'in-progress', 'in-review', 'in-test', 'in-prod'],
     change_colors=array['15', '14', '10', '11', '12', '13'],
     change_types=array['feature', 'fix', 'refactor', 'upgrade', 'chore', 'docs', 'test', 'ci', 'security', 'migration', 'revert', 'spike'];

insert into public.user (
    id, email, name, token
) values (
    0, 'vitodivilla@gmail.com', 'superadmin', 'xxx-token'
) on conflict (id) do update set
    email='vitodivilla@gmail.com',
    name='superadmin',
    token='xxx-token';

commit;
