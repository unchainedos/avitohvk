alter table chain_deals
    add column creator_id uuid not null references users(id);

create index idx_chain_deals_creator on chain_deals (creator_id);
