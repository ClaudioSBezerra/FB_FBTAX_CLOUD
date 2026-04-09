-- Migration 105: atualiza destination_url dos produtos do portal
UPDATE portal.pt_products SET destination_url = 'https://apuracao.fbtax.cloud'  WHERE name = 'Apuração Assistida';
UPDATE portal.pt_products SET destination_url = 'https://simulador.fbtax.cloud' WHERE name = 'Simulador Fiscal';
UPDATE portal.pt_products SET destination_url = 'https://smartpick.fbtax.cloud' WHERE name = 'SmartPick';
-- Farol: sem URL por enquanto
UPDATE portal.pt_products SET destination_url = '' WHERE name = 'Farol';
