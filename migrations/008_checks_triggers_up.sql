alter table users
    add constraint chk_users_rating_non_negative  check (rating >= 0),
    add constraint chk_users_decline_non_negative check (decline_count >= 0);

alter table items
    add constraint chk_items_quantity_positive check (quantity > 0);

alter table transactions
    add constraint chk_transactions_quantity_positive check (quantity > 0);

alter table chain_deals
    add constraint chk_chain_deals_participants check (participants >= 2),
    add constraint chk_chain_deals_deadline_after_created check (deadline_at > created_at),
    add constraint chk_chain_deals_updated_after_created check (updated_at >= created_at);

create or replace function set_updated_at()
returns trigger as $$
begin
    new.updated_at := now();
    return new;
end;
$$ language plpgsql;

create trigger trg_chain_deals_updated_at
    before update on chain_deals
    for each row
    execute function set_updated_at();

create trigger trg_chain_deals_transactions_updated_at
    before update on chain_deal_transactions
    for each row
    execute function set_updated_at();