-- Run with psql -v ON_ERROR_STOP=1 -f db/tests/change_history.sql after init.sql.
begin;

do
$$
declare
    _project_id bigint;
    _change_id bigint;
    _missing_id bigint;
    _doc_type text;
    _version smallint := 0;
begin
    insert into public.project (name) values ('change history test') returning id into _project_id;
    select public.fn_change_insert(_project_id, gen_random_uuid(), 'history', 'initial') into _change_id;
    select coalesce(max(id), 0) + 1 from public.change into _missing_id;

    foreach _doc_type in array array['brief', 'spec', 'pr'] loop
        -- Missing IDs are no-ops, without a phantom history record.
        execute format('call public.sp_change_%s_update($1, $2, $3)', _doc_type)
            using _missing_id, 'missing', false;
        assert not exists (select from public.change_history where id = _missing_id),
            format('%s update created history for a missing change', _doc_type);

        -- Valid updates still increment the shared version and record the new body.
        execute format('call public.sp_change_%s_update($1, $2, $3)', _doc_type)
            using _change_id, 'updated ' || _doc_type, true;
        _version := _version + 1;
        assert (select version = _version from public.change where id = _change_id),
            format('%s update did not increment version', _doc_type);
        assert exists (
            select from public.change_history h join public.change c on c.id = h.id
            where h.id = _change_id and h.version = _version and h.doc_type = _doc_type
                and h.body = 'updated ' || _doc_type and h.agent_edit and h.modified = c.modified
        ), format('%s history does not match the update', _doc_type);
    end loop;

    delete from public.change where id = _change_id;
    foreach _doc_type in array array['brief', 'spec', 'pr'] loop
        execute format('call public.sp_change_%s_update($1, $2, $3)', _doc_type)
            using _change_id, 'deleted', false;
    end loop;
    assert (select count(*) = 4 from public.change_history where id = _change_id),
        'updates to a deleted change must preserve existing history';
end;
$$;

rollback;
