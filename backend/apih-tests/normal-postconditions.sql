-- DB evidence: normal lifecycle must leave outage fixtures unchanged.
\ir outage-postconditions.sql
DO $$
BEGIN
 -- P3 deletes the parent but must retain exact append-only document history.
 IF EXISTS (SELECT 1 FROM public.change WHERE id=2204)
 OR (SELECT count(*) FROM public.doc WHERE ref_id=2204 AND ref_table='change') <> 2
 OR NOT EXISTS (SELECT 1 FROM public.doc WHERE id=2401 AND ref_id=2204 AND ref_table='change' AND body='Historic' AND NOT current)
 OR NOT EXISTS (SELECT 1 FROM public.doc WHERE id=2402 AND ref_id=2204 AND ref_table='change' AND body='Current' AND current) THEN
  RAISE EXCEPTION 'normal deletion did not retain append-only documents';
 END IF;
END $$;
