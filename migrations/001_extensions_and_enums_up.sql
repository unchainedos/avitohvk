create extension if not exists pgcrypto;

create type link_status as enum (
    'PENDING', 'ACCEPTED', 'DECLINED' , 'WAITING_FOR_THE_REQUIRED_USER'
);

create type deal_status as enum(
    'PENDING' , 'CONFIRMED', 'COMPLETED', 'CANCELLED'
);

