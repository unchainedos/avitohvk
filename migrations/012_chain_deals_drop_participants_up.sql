alter table chain_deals
    drop constraint chk_chain_deals_participants;

alter table chain_deals
    drop column participants;
