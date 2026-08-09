alter table items
    add column locked_by_deal_id uuid references chain_deals(id);

create index idx_items_locked_by_deal on items(locked_by_deal_id);
