# Ambiente de desenvolvimento local (WSL Ubuntu)

Roda a stack inteira em `localhost`: PostgreSQL nativo + backend Go + frontend Vite.
Sem Docker — o WSL já dá tudo que é preciso.

## Primeira vez

```bash
make setup
```

Instala o PostgreSQL se faltar, sobe o serviço, cria o banco `fbtax_cloud`, gera
`backend/.env` e instala as dependências. **Pede sua senha do sudo** nas etapas de
instalação e de start do serviço — o restante roda sem privilégio.

É idempotente: pode rodar de novo à vontade. Se `backend/.env` já existir, ele é
preservado, nunca sobrescrito.

## No dia a dia

```bash
make dev
```

| | |
|---|---|
| Frontend | http://localhost:3086 |
| Backend  | http://localhost:8086/api/health |
| Termos de Entrega | http://localhost:3086/admin/financeiro/entregas |
| Login | `claudio_bezerra@hotmail.com` / `123456` |

`Ctrl+C` derruba os dois processos. Os logs ficam em `dev/logs/`.

O login sai da migration `021_ensure_admin_user.sql`, que garante esse usuário com
papel `admin` toda vez que as migrations rodam.

## Como as peças se encaixam

```
navegador :3086
      │
      ├─ /            → Vite dev server (HMR do React)
      └─ /api/*       → proxy do Vite  ─→  backend Go :8086
                                              │
                                              └─→ PostgreSQL :5432
                                                   banco fbtax_cloud
```

O proxy está em `frontend/vite.config.ts` e aponta para `VITE_API_TARGET`
(padrão `http://localhost:8086`). Em produção quem faz esse roteamento é o
Traefik — por isso o frontend nunca precisa saber a URL da API.

## Migrations

O backend aplica as migrations pendentes **no boot**, lendo `backend/migrations/*.sql`
e registrando o que já rodou em `schema_migrations`. Não existe comando separado:
subir o backend já migra.

```bash
make migrations   # o que já foi aplicado
make psql         # console do banco
```

Se uma migration falhar, o backend **sobe assim mesmo** e registra `Migration FAILED`
no log — o erro não derruba o processo, mas a tela que depende da tabela vai falhar.
Vale conferir `dev/logs/backend.log` depois de adicionar uma migration nova.

## Comandos

```
make help        lista tudo
make setup       prepara o ambiente
make dev         sobe backend + frontend
make backend     só o backend
make frontend    só o frontend
make test        go vet + go test + typecheck do frontend
make build       compila os dois como em produção
make psql        console do banco
make migrations  migrations aplicadas
make db-reset    APAGA e recria o banco (pede confirmação)
make clean       remove binários, dist e logs
```

## Problemas comuns

**`pg_isready` diz que está fora do ar depois de reiniciar o Windows.**
O WSL não preserva serviços entre sessões. `make dev` já tenta subir sozinho; para
não depender disso, habilite de vez:

```bash
sudo systemctl enable postgresql
```

**Porta 8086 ou 3086 ocupada.**

```bash
ss -lptn 'sport = :8086'
```

**Autenticação falhou no banco.** A `DATABASE_URL` conecta por TCP em `localhost`,
o que exige senha — não o `peer` que o psql usa por socket. Rode `make setup` de
novo, que ele reaplica a senha do papel `postgres`.

**Frontend abre mas as telas ficam vazias.** Quase sempre é o backend fora do ar: o
proxy do Vite responde, mas a API não. Confira `http://localhost:8086/api/health`.
