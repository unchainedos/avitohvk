-- Создание таблицы items с поддержкой полнотекстового поиска
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
  
  search_vector tsvector generated always as (
    setweight(to_tsvector('russian', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('russian', coalesce(description, '')), 'B') ||
    setweight(to_tsvector('russian', coalesce(category, '')), 'C')
  ) stored,

  constraint fk_items_user_author foreign key (author_id) references users(id),
  constraint fk_items_user_holder foreign key (holder_id) references users(id)
);

create index idx_items_author_id on items(author_id);
create index idx_items_holder_id on items(holder_id);
create index idx_items_category on items(category);
create index idx_items_locked_holder on items(holder_id, is_locked);

create index idx_items_search_vector on items using gin(search_vector);

create index idx_items_search_vector_rank on items using gin(search_vector) 
where search_vector is not null;