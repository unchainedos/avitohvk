drop trigger if exists trg_chain_deals_transactions_updated_at on chain_deal_transactions;
drop trigger if exists trg_chain_deals_updated_at on chain_deals;

drop function if exists set_updated_at();

alter table chain_deals
    drop constraint if exists chk_chain_deals_updated_after_created,
    drop constraint if exists chk_chain_deals_deadline_after_created,
    drop constraint if exists chk_chain_deals_participants;

alter table transactions
    drop constraint if exists chk_transactions_quantity_positive;

alter table items
    drop constraint if exists chk_items_quantity_positive;

alter table users
    drop constraint if exists chk_users_decline_non_negative,
    drop constraint if exists chk_users_rating_non_negative;