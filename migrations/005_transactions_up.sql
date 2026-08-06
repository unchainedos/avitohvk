create table transactions (
  id         uuid primary key default gen_random_uuid(), 
  item_id    uuid not null,
  from_user  uuid not null,
  to_user    uuid not null,
  quantity   numeric not null default 1,
  created_at timestamptz not null default now(),

  constraint fk_transactions_item_id foreign key (item_id) references items(id) ,
  constraint fk_transactions_from_user foreign key (from_user) references users(id),
  constraint fk_transactions_to_user foreign key (to_user) references users(id)
);

create index idx_transactions_item_id_created_at on transactions(item_id, created_at);
create index idx_transactions_from_user on transactions(from_user);
create index idx_transactions_to_user on transactions(to_user);