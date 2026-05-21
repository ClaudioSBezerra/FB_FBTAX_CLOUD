package handlers

// inter.go — Os endpoints /api/financeiro/inter/* foram substituídos pela
// arquitetura multi-banco em handlers/bank_config.go:
//
//   GET  /api/financeiro/bancos/status    → BancosStatusHandler
//   PUT  /api/financeiro/bancos/config    → BancosConfigHandler
//   POST /api/financeiro/bancos/sync      → BancosSyncHandler
//   GET  /api/financeiro/bancos/providers → BancosProvidersHandler
