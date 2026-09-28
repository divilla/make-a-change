-- Owned disposable cluster only. IDs >= 1000 are reserved for fixed fixtures;
-- generated identity IDs remain small and are captured by the serial workflow.
INSERT INTO public.config(slug,project_docs,epic_docs,change_docs,change_phases,change_colors,change_types)
VALUES ('p2-custom',ARRAY['vision','notes'],ARRAY['design','risk'],ARRAY['analysis','decision'],
        ARRAY['discover','ship'],ARRAY['#123456','#abcdef'],ARRAY['experiment','maintenance']);
INSERT INTO public.project(id,name,config,last_ref,created,modified) VALUES
 (1001,'Custom project','p2-custom',42,'2026-01-01','2026-01-01'),
 (1002,'Default project','default',0,'2026-01-01','2026-01-01'),
 (1003,'Missing config project','p2-absent',0,'2026-01-01','2026-01-01'),
 (1004,'Empty project','default',0,'2026-01-01','2026-01-01'),
 (1005,'Epic parent','default',0,'2026-01-01','2026-01-01'),
 (1006,'Change parent','default',1,'2026-01-01','2026-01-01');
INSERT INTO public.epic(id,project_id,name,created,modified) VALUES
 (1101,1005,'Empty epic','2026-01-01','2026-01-01'),
 (1102,1001,'Half complete','2026-01-02','2026-01-02'),
 (1103,1001,'Zero testcases','2026-01-02','2026-01-02');
INSERT INTO public.change(id,project_id,epic_id,title,change_phase) VALUES
 (1201,1006,NULL,'Project dependency','backlog'),
 (1202,1001,1102,'Epic dependency','discover');
INSERT INTO public.testcase(id,change_id,scenario,done) VALUES
 (1301,1202,'Done case',true),(1302,1202,'Pending case',false);

-- P3 isolated fixtures. P2 rows and relationships remain unchanged.
INSERT INTO public.project(id,name,config,created,modified) VALUES
 (2001,'P3 default','default','2025-01-01','2025-01-01'),
 (2002,'P3 custom','p2-custom','2025-01-01','2025-01-01'),
 (2003,'P3 missing config','p3-absent','2025-01-01','2025-01-01');
INSERT INTO public.epic(id,project_id,name) VALUES
 (2101,2001,'P3 matching epic'),(2102,2002,'P3 foreign epic');
INSERT INTO public.change(id,project_id,title,change_phase,created,modified) VALUES
 (2201,2001,'P3 default change','backlog','2025-01-01','2025-01-01'),
 (2202,2002,'P3 custom change','discover','2025-01-01','2025-01-01'),
 (2203,2003,'P3 missing config change','backlog','2025-01-01','2025-01-01'),
 (2204,2001,'P3 deletable change','backlog','2025-01-01','2025-01-01'),
 (2205,2001,'P3 no documents','backlog','2025-01-01','2025-01-01'),
 (2206,2001,'P3 wide counts','backlog','2025-01-01','2025-01-01'),
 (2207,2001,'P3 duplicate current artifacts','backlog','2025-01-01','2025-01-01');
INSERT INTO public.testcase(id,change_id,scenario,done)
 SELECT 230000+n,2206,'Wide count case '||n,n<=40000 FROM generate_series(1,50000) AS n;
INSERT INTO public.doc(id,ref_id,ref_table,doc_type,body,current) VALUES
 (2401,2204,'change','brief','Historic',false),
 (2402,2204,'change','brief','Current',true),
 (2403,2203,'change','spec','**No config needed**',true),
 (2404,2207,'change','spec','Old duplicate',true),
 (2405,2207,'change','spec','Latest duplicate',true),
 (2406,2207,'project','pr','Wrong table',true),
 (2407,2207,'change','pr','Historical PR',false);
