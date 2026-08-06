create table users (
    id            uuid         primary key default gen_random_uuid(),
    username      varchar      not null unique,
    password_hash varchar      not null,
    rating        numeric      not null default 0,
    decline_count int          not null default 0,
    is_banned     boolean      not null default false,
    tg            varchar,
    phone_number  varchar,
    email         varchar      unique,
    created_at    timestamptz  not null default now()
);