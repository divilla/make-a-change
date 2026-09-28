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
  select public.fn_doc_insert(_ref.id,_ref.kind,'brief','Identical',false) into _id;
  select public.fn_doc_insert(_ref.id,_ref.kind,'brief','Identical',false) into _next;
  assert _next>_id, 'each insert returns its distinct new doc ID';
  assert (select not current and body='Identical' from public.doc where id=_id), 'previous raw doc retained';
  assert (select current and ref_id=_ref.id and ref_table=_ref.kind from public.doc where id=_next), 'new current reference';
  assert (select count(*)=1 from public.doc where ref_id=_ref.id and ref_table=_ref.kind and doc_type='brief' and current), 'exactly one current row';
  assert (select created_by=0 and updated_by=0 from public.doc where id=_next), 'attribution keeps SQL defaults until sessions';
 end loop;
 begin
  perform public.fn_config_delete('default');
  raise exception 'referenced config was deleted';
 exception when foreign_key_violation then null;
 end;
 assert exists(select 1 from public.config where slug='default'), 'referenced config retained';
 assert not public.fn_config_delete('nonexistent-config'), 'missing delete result';
 insert into public.config(slug,project_docs,epic_docs,change_docs,change_phases,change_colors,change_types)
 values('disposable',array[]::text[],array[]::text[],array[]::text[],array[]::text[],array[]::text[],array[]::text[]);
 assert public.fn_config_delete('disposable'), 'unreferenced config deleted';
 delete from public.change where id=_change;
 assert public.fn_doc_insert(_change,'change','brief','Absent',false) is null, 'absent parent is not resurrected';
 assert exists(select 1 from public.doc where ref_id=_change and ref_table='change'), 'deleted parent retains history';
end;
$$;
rollback;
