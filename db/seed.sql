begin;

truncate table public.change_phase;
truncate table public.change_type;
truncate table public.config;

insert into public.config (
    slug,
    flow_stages,
    flow_stages_help,
    flow_stage_modes_default,
    flow_stage_entry_scripts,
    flow_stage_prompts,
    flow_stage_exit_scripts,
    stage_modes,
    stage_modes_help,
    task_statuses,
    task_statuses_help,
    task_steps,
    task_steps_help,
    change_phases,
    change_types,
    change_docs
) values (
'default',

    array[
        'artifact', 'spec', 'ready', 'docs', 'code', 'polish', 'pr',
        'review', 'fix', 'sync', 'merge', 'stage', 'master'
    ],

    array[
        'capture or refine the initial change definition',
        'create the implementation-ready spec',
        'mark the change as ready for execution',
        'update required documentation before implementation',
        'implement the change',
        'user-guided refinement after coding',
        'create or update the pull request',
        'review the PR/change against the spec',
        'address review findings',
        'align spec, QA cases, and docs with final code changes',
        'merge the change branch',
        'promote/merge into stage branch',
        'promote/merge into master branch'
    ],

    array[
        'prompt', 'exec', 'exec', 'exec', 'exec', 'prompt', 'exec',
        'exec', 'exec', 'exec', 'exec', 'exec', 'exec'
    ],

    array[
        '', '', '', '', '', '', '',
        '', '', '', '', '', ''
    ],

    array[
        '', '', '', '', '', '', '',
        '', '', '', '', '', ''
    ],

    array[
        '', '', '', '', '', '', '',
        '', '', '', '', '', ''
    ],

    array['skip', 'prompt', 'exec'],

    array[
        'stage will not execute',
        'stage will run an interactive session',
        'stage will run an automated agent'
    ],

    array[
        'queued', 'running', 'paused', 'stopped', 'waiting', 'completed', 'failed'
    ],

    array[
        'task is waiting to start',
        'task is actively executing',
        'task is temporarily paused',
        'task was manually stopped',
        'task is waiting for input',
        'task finished successfully',
        'task finished with an error'
    ],

    array['none', 'entry', 'prompt', 'agent', 'exit', 'done'],

    array[
        'task has not started yet',
        'entry script is executing',
        'interactive session is running',
        'automated agent is executing',
        'exit script is executing',
        'task has finished'
    ],

    array['backlog', 'progress', 'review', 'staging', 'production', 'rejected'],

    array['feature', 'fix', 'refactor', 'upgrade', 'chore', 'docs', 'test', 'ci', 'security', 'migration', 'revert', 'spike'],

    array['def', 'spec', 'pr']
);

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

commit;
