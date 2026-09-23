pg_dump -c -U postgres -d changes_test -h localhost | gzip > changes_test`date +%Y-%m-%d"_"%H_%M_%S`.sql.gz
