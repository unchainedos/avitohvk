create table items (
  id            uuid primary key default gen_random_uuid(),
  author_id     uuid not null,
  holder_id     uuid not null,
  title         varchar not null,
  description   text,
  image_url     varchar,
  category      varchar,
  unit          varchar,
  quantity      numeric not null default 1,
  is_locked     boolean not null default false,
  created_at    timestamptz not null default now(),

  constraint fk_items_user_author foreign key (author_id) references users(id),
  constraint fk_items_user_holder foreign key (holder_id) references users(id)
);

create index idx_author_id on items(author_id);
create index idx_holder_id on items(holder_id);
create index idx_category on items(category);
create index idx_locked_holder on items(holder_id, is_locked);

