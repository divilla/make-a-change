create table public.uri
(
    url      text              not null
        constraint uri_pk
            primary key,
    title    text,
    content  text,
    children integer default 0 not null
);

alter table public.uri
    owner to postgres;

create table public.category
(
    url  text                      not null
        constraint category_pk
            primary key,
    id   text                      not null,
    code text                      not null,
    data jsonb default '{}'::jsonb not null
);

alter table public.category
    owner to postgres;
