-- Run only on an owned disposable database after init.sql and seed.sql.
begin;
do $$
declare
 _project bigint;
 _epic bigint;
 _change bigint;
 _id bigint;
 _next bigint;
 _ref record;
begin
 insert into public.project(name) values('Doc contract') returning id into _project;
 insert into public.epic(project_id,name) values(_project,'Doc contract') returning id into _epic;
 select public.fn_change_insert(_project,gen_random_uuid(),'Doc contract','Initial') into _change;
 for _ref in select * from (values ('project',_project),('epic',_epic),('change',_change)) as r(kind,id) loop
  select public.fn_doc_insert(_ref.kind,_ref.id,'brief','Identical',false) into _id;
  select public.fn_doc_insert(_ref.kind,_ref.id,'brief','Identical',false) into _next;
  assert _next>_id, 'each insert returns its distinct new doc ID';
  assert exists(select 1 from public.doc where id=_id and body='Identical'), 'previous raw doc retained';
  assert not exists(select 1 from public.doc_active where doc_id=_id), 'previous raw doc is not active';
  assert exists(select 1 from public.doc_active where ref_id=_ref.id and ref_table=_ref.kind and doc_type='brief' and doc_id=_next), 'new active reference';
  assert (select count(*)=1 from public.doc_active where ref_id=_ref.id and ref_table=_ref.kind and doc_type='brief'), 'exactly one active row';
  assert (select created_by=0 and updated_by=0 from public.doc where id=_next), 'attribution keeps SQL defaults until sessions';
 end loop;
 begin
  delete from public.config where slug='default';
  raise exception 'referenced config was deleted';
 exception when foreign_key_violation then null;
 end;
 assert exists(select 1 from public.config where slug='default'), 'referenced config retained';
 delete from public.config where slug='nonexistent-config';
 assert not found, 'missing config deletion affects no rows';
 insert into public.config(slug,project_docs,epic_docs,change_docs,change_phases,change_colors,change_types)
 values('disposable',array[]::text[],array[]::text[],array[]::text[],array[]::text[],array[]::text[],array[]::text[]);
 delete from public.config where slug='disposable';
 assert found, 'unreferenced config deletion affects a row';
 assert not exists(select 1 from public.config where slug='disposable'), 'unreferenced config deleted';
 assert public.fn_doc_delete(_next)=_next, 'delete returns the document ID';
 assert (select deleted_by=0 and deleted_at=now() from public.doc where id=_next), 'soft deletion records attribution and time';
 assert not exists(select 1 from public.doc_active where doc_id=_next), 'deleted document is no longer active';
 assert public.fn_doc_delete(-1) is null, 'missing document delete result';
 _id := public.fn_doc_comment_insert(_project,'project','Comment',false);
 assert public.fn_doc_comment_update(_id,'Edited comment')=_id, 'comment update returns the document ID';
 assert (select body='Edited comment' and updated_by=0 and updated_at=now() from public.doc where id=_id), 'comment body and attribution updated';
 assert not exists(select 1 from public.doc_active where doc_id=_id), 'comments have no active reference';
 assert public.fn_doc_comment_update(_id,'')=_id, 'empty comment update is supported';
 assert public.fn_doc_comment_update(_next,'Not a comment') is null, 'comment update excludes other document types';
 assert public.fn_doc_comment_update(-1,'Missing') is null, 'missing comment update result';
end;
$$;
rollback;
