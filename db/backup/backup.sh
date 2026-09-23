pg_dump -c -U postgres -d changes -h localhost | gzip > changes_`date +%Y-%m-%d"_"%H_%M_%S`.sql.gz
