-- Migration 107: atualiza descrição do Farol
UPDATE portal.pt_products SET description = 'Gestão de Objetivos de Vendas e Resultados' WHERE name = 'Farol';
