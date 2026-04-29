-- Migration 106: ativa URL do Farol
UPDATE portal.pt_products SET destination_url = 'https://farol.fbtax.cloud' WHERE name = 'Farol';
