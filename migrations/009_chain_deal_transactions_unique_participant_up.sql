alter table chain_deal_transactions
    add constraint uq_cdt_deal_participant unique (deal_id, participant_id);
