alter table chain_deals
    add column negotiation_window_seconds integer not null,
    alter column deadline_at drop not null;

alter table chain_deals
    drop constraint chk_chain_deals_deadline_after_created,
    add constraint chk_chain_deals_deadline_after_created
        check (deadline_at is null or deadline_at > created_at);

alter table chain_deals
    add constraint chk_chain_deals_negotiation_window_positive
        check (negotiation_window_seconds > 0);
