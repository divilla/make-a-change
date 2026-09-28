-- DB evidence only: these SQL assertions earn no Go statement coverage.
DO $$
BEGIN
 IF (SELECT count(*) FROM public.project WHERE id IN (3001,3002)) <> 2
 OR NOT EXISTS (SELECT 1 FROM public.project WHERE id=3001 AND name='Outage parent' AND config='default' AND last_ref=0 AND created='2024-01-01' AND modified=created)
 OR NOT EXISTS (SELECT 1 FROM public.project WHERE id=3002 AND name='Outage deletable project' AND config='default' AND last_ref=0 AND created='2024-01-01' AND modified=created)
 OR EXISTS (SELECT 1 FROM public.project WHERE name='Outage forbidden insert') THEN
  RAISE EXCEPTION 'outage project write persisted or retained parent changed';
 END IF;
 IF (SELECT count(*) FROM public.epic WHERE project_id=3001) <> 2
 OR NOT EXISTS (SELECT 1 FROM public.epic WHERE id=3101 AND project_id=3001 AND name='Outage epic' AND created='2024-01-01' AND modified=created)
 OR NOT EXISTS (SELECT 1 FROM public.epic WHERE id=3102 AND project_id=3001 AND name='Outage deletable epic' AND created='2024-01-01' AND modified=created) THEN
  RAISE EXCEPTION 'outage epic insert/update/delete persisted';
 END IF;
 IF (SELECT count(*) FROM public.change WHERE project_id=3001) <> 2
 OR NOT EXISTS (SELECT 1 FROM public.change WHERE id=3201 AND project_id=3001 AND epic_id=3101 AND title='Outage change' AND open AND pr_url='' AND change_phase='backlog' AND change_types='{}' AND created='2024-01-01' AND modified=created)
 OR NOT EXISTS (SELECT 1 FROM public.change WHERE id=3202 AND project_id=3001 AND epic_id IS NULL AND title='Outage deletable change' AND open AND pr_url='' AND created='2024-01-01' AND modified=created) THEN
  RAISE EXCEPTION 'outage change write persisted';
 END IF;
 IF (SELECT count(*) FROM public.testcase WHERE change_id IN (3201,3202)) <> 2
 OR NOT EXISTS (SELECT 1 FROM public.testcase WHERE id=3301 AND change_id=3201 AND scenario='Original scenario' AND NOT done AND created='2024-01-01')
 OR NOT EXISTS (SELECT 1 FROM public.testcase WHERE id=3302 AND change_id=3201 AND scenario='Retained delete target' AND done AND created='2024-01-01' AND modified=created) THEN
  RAISE EXCEPTION 'outage testcase write persisted or recovery fixture not restored';
 END IF;
 IF (SELECT count(*) FROM public.doc WHERE ref_table='change' AND ref_id IN (3201,3202)) <> 1
 OR NOT EXISTS (SELECT 1 FROM public.doc WHERE id=3401 AND ref_id=3201 AND ref_table='change' AND doc_type='brief' AND body='**Original document**' AND current AND NOT agent_edit AND created='2024-01-01') THEN
  RAISE EXCEPTION 'outage document mutation persisted';
 END IF;
END $$;
