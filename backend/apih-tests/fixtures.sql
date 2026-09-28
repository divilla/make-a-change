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
