create table chain_deals (
    id           uuid        primary key default gen_random_uuid(),
    root_item_id uuid        not null,
    status       deal_status not null default 'PENDING',
    participants int         not null default 2,
    deadline_at  timestamptz not null,
    created_at   timestamptz not null default now(),
    updated_at   timestamptz not null default now(),

    constraint fk_chain_deals_root_item foreign key (root_item_id) references items(id)
);

create index idx_chain_deals_root_item on chain_deals (root_item_id);
create index idx_chain_deals_status    on chain_deals (status);
create index idx_chain_deals_deadline  on chain_deals (deadline_at);