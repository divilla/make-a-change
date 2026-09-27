-- Run with psql -v ON_ERROR_STOP=1 -f db/tests/change_history.sql after init.sql.
-- Documents now live in doc, with one row per submitted body.
begin;
\ir fixtures.sql

do $$
declare
    _project_id bigint := pg_temp.create_project('document test');
    _epic_id bigint;
    _change_id bigint;
    _entity record;
    _doc_type text;
    _before bigint;
    _modified_matches boolean;
begin
    insert into public.epic (project_id, name) values (_project_id, 'epic') returning id into _epic_id;
    _change_id := public.fn_change_insert(_project_id, gen_random_uuid(), 'documents', 'initial');
    assert exists (
        select from public.doc d join public.change c on c.id = d.ref_id
        where c.id = _change_id and d.ref_table = 'change' and d.doc_type = 'brief'
            and d.body = 'initial' and not d.agent_edit and d.created = c.created
    ), 'creation records the initial brief';

    for _entity in select * from (values
        ('project', _project_id), ('epic', _epic_id), ('change', _change_id)
    ) as entities(kind, id) loop
        foreach _doc_type in array array['brief', 'spec', 'pr', 'comment'] loop
            select count(*) into _before from public.doc;
            execute format('call public.sp_%s_doc_set($1, $2, $3, $4)', _entity.kind)
                using -1::bigint, _doc_type, 'missing', false;
            assert (select count(*) = _before from public.doc), 'missing parents do not create documents';

            execute format('update public.%I set modified = ''2000-01-01'' where id = $1', _entity.kind)
                using _entity.id;
            execute format('call public.sp_%s_doc_set($1, $2, $3, $4)', _entity.kind)
                using _entity.id, _doc_type, 'first body', true;
            execute format('call public.sp_%s_doc_set($1, $2, $3, $4)', _entity.kind)
                using _entity.id, _doc_type, 'second body', false;
            assert (select count(*) = _before + 2 from public.doc), 'each submission appends a document';
            assert exists (select from public.doc where ref_id = _entity.id and ref_table = _entity.kind
                and doc_type = _doc_type and body = 'first body' and agent_edit), 'earlier body is retained';
            assert exists (select from public.doc where ref_id = _entity.id and ref_table = _entity.kind
                and doc_type = _doc_type and body = 'second body' and not agent_edit), 'new body is retained';
            execute format('select modified = now() from public.%I where id = $1', _entity.kind)
                into strict _modified_matches using _entity.id;
            assert _modified_matches, 'document writes update the parent timestamp';
            assert exists (select from public.doc where ref_id = _entity.id and ref_table = _entity.kind
                and doc_type = _doc_type and body = 'second body' and created = now()),
                'document timestamp matches the parent update';
        end loop;
    end loop;

    select count(*) into _before from public.doc;
    delete from public.change where id = _change_id;
    delete from public.epic where id = _epic_id;
    delete from public.project where id = _project_id;
    call public.sp_change_doc_set(_change_id, 'brief', 'deleted', false);
    call public.sp_epic_doc_set(_epic_id, 'brief', 'deleted', false);
    call public.sp_project_doc_set(_project_id, 'brief', 'deleted', false);
    assert (select count(*) = _before from public.doc), 'deletion preserves documents; later writes are no-ops';
end;
$$;
rollback;
