FROM postgres:latest
COPY entrypoint/* /docker-entrypoint-initdb.d/
RUN chown -R postgres:postgres /docker-entrypoint-initdb.d
