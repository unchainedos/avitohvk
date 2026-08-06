create table chain_deal_transactions (
    deal_id        uuid        not null,
    transaction_id uuid        not null,
    participant_id uuid        not null,
    status         link_status not null default 'PENDING',
    updated_at     timestamptz not null default now(),

    constraint fk_cdt_deal        foreign key (deal_id)        references chain_deals(id)  on delete cascade,
    constraint fk_cdt_transaction foreign key (transaction_id) references transactions(id),
    constraint fk_cdt_participant foreign key (participant_id) references users(id),

    primary key (deal_id, transaction_id)
);

create index idx_cdt_deal        on chain_deal_transactions (deal_id);
create index idx_cdt_participant on chain_deal_transactions (participant_id);