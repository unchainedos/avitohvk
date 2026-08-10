alter table chain_deals
    add column participants int not null default 2;

alter table chain_deals
    add constraint chk_chain_deals_participants check (participants >= 2);
