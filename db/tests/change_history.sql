-- Run with psql -v ON_ERROR_STOP=1 -f db/tests/change_history.sql after init.sql and seed.sql.
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
    _first bigint;
    _second bigint;
begin
    insert into public.epic (project_id, name) values (_project_id, 'epic') returning id into _epic_id;
    _change_id := public.fn_change_insert(_project_id, gen_random_uuid(), 'documents', 'initial');
    assert exists (
        select from public.doc d join public.change c on c.id = d.ref_id
        where c.id = _change_id and d.ref_table = 'change' and d.doc_type = 'brief'
            and d.body = 'initial' and not d.agent_edit and d.created_at = c.created_at
    ), 'creation records the initial brief';

    for _entity in select * from (values
        ('project', _project_id), ('epic', _epic_id), ('change', _change_id)
    ) as entities(kind, id) loop
        foreach _doc_type in array array['brief', 'spec', 'pr', 'comment'] loop
            select count(*) into _before from public.doc;
            if _doc_type = 'comment' then
                _first := public.fn_doc_comment_insert(_entity.id, _entity.kind, 'first body', true);
                _second := public.fn_doc_comment_insert(_entity.id, _entity.kind, 'second body', false);
                assert not exists (select from public.doc_active
                    where ref_id = _entity.id and ref_table = _entity.kind and doc_type = 'comment'),
                    'comments do not replace an active document';
            else
                _first := public.fn_doc_insert(_entity.kind, _entity.id, _doc_type, 'first body', true);
                _second := public.fn_doc_insert(_entity.kind, _entity.id, _doc_type, 'second body', false);
                assert exists (select from public.doc_active
                    where ref_id = _entity.id and ref_table = _entity.kind and doc_type = _doc_type and doc_id = _second),
                    'active reference selects the latest document';
                assert exists (select from public.vw_doc_active
                    where ref_id = _entity.id and ref_table = _entity.kind and doc_type = _doc_type
                        and body = 'second body' and not agent_edit),
                    'active view returns the latest body';
            end if;
            assert _second > _first, 'each submission returns a distinct document ID';
            assert (select count(*) = _before + 2 from public.doc), 'each submission appends a document';
            assert exists (select from public.doc where ref_id = _entity.id and ref_table = _entity.kind
                and doc_type = _doc_type and body = 'first body' and agent_edit), 'earlier body is retained';
            assert exists (select from public.doc where ref_id = _entity.id and ref_table = _entity.kind
                and doc_type = _doc_type and body = 'second body' and not agent_edit), 'new body is retained';
            assert exists (select from public.doc where ref_id = _entity.id and ref_table = _entity.kind
                and doc_type = _doc_type and body = 'second body' and created_at = now()),
                'document timestamp matches the submission transaction';
        end loop;
    end loop;

    select count(*) into _before from public.doc;
    delete from public.change where id = _change_id;
    delete from public.epic where id = _epic_id;
    delete from public.project where id = _project_id;
    assert (select count(*) = _before from public.doc), 'parent deletion preserves document history';
end;
$$;
rollback;
