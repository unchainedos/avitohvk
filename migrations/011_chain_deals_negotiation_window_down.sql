alter table chain_deals
    drop constraint chk_chain_deals_negotiation_window_positive;

alter table chain_deals
    drop constraint chk_chain_deals_deadline_after_created,
    add constraint chk_chain_deals_deadline_after_created
        check (deadline_at > created_at);

alter table chain_deals
    drop column negotiation_window_seconds,
    alter column deadline_at set not null;
