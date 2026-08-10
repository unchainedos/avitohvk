create table wishes(
    id             uuid primary key default gen_random_uuid(),
    user_id        uuid not null,
    title          varchar not null,
    description    text,
    created_at     timestamptz not null default now(),

    constraint fk_wishes_user foreign key (user_id) references users(id) 
);

create index idx_wish_user_id on wishes(user_id);
create index idx_wish_title on wishes(title);

create table wish_items(
    wish_id uuid not null,
    item_id uuid not null,

    constraint fk_wishes_items_wish_id foreign key (wish_id) references wishes(id) on delete cascade,
    constraint fk_wishes_items_item_id foreign key (item_id) references items(id) on delete cascade,

    primary key (wish_id, item_id)
);

create index idx_wish_items_item_id on wish_items(item_id);