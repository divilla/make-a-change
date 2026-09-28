-- DB evidence only, before any recovery writes. No Go coverage credit.
\ir recovery-postconditions.sql
DO $$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM public.testcase WHERE id=3301 AND modified=created) THEN
  RAISE EXCEPTION 'failed testcase update changed timestamp';
 END IF;
END $$;
