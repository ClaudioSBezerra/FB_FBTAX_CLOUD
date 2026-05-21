package services

// ProviderConfig contém todas as credenciais possíveis de um banco.
// Cada provider usa apenas os campos que precisa.
type ProviderConfig struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	CertPEM       string `json:"cert_pem,omitempty"`       // base64 ou PEM bruto
	KeyPEM        string `json:"key_pem,omitempty"`        // base64 ou PEM bruto
	ContaCorrente string `json:"conta_corrente,omitempty"` // Inter: x-conta-corrente
	Agencia       string `json:"agencia,omitempty"`        // Itaú: agência
	Conta         string `json:"conta,omitempty"`          // Itaú: número da conta
	BaseURL       string `json:"base_url,omitempty"`       // override do endpoint
}

// BankSaldo representa o saldo de uma conta bancária.
type BankSaldo struct {
	Disponivel float64
	Bloqueado  float64
}

// BankTransacao é a representação normalizada de um lançamento bancário.
type BankTransacao struct {
	IDExterno     string  // identificador único no banco (para deduplicação)
	DataTransacao string  // YYYY-MM-DD
	Descricao     string
	Valor         float64 // sempre positivo
	Tipo          string  // "credito" | "debito"
	Categoria     string  // tipo de operação conforme o banco
}

// BankProvider é a interface que todos os bancos implementam.
type BankProvider interface {
	GetSaldo(cfg ProviderConfig) (*BankSaldo, error)
	GetExtrato(cfg ProviderConfig, dataInicio, dataFim string) ([]BankTransacao, error)
}

// ── Registry ─────────────────────────────────────────────────────────────────

var providerRegistry = map[string]BankProvider{
	"inter": &InterProvider{},
	"itau":  &ItauProvider{},
}

// GetProvider retorna o provider para o nome de banco informado.
func GetProvider(nome string) (BankProvider, bool) {
	p, ok := providerRegistry[nome]
	return p, ok
}

// ProvidersDisponiveis retorna os nomes de todos os provedores registrados.
func ProvidersDisponiveis() []string {
	names := make([]string, 0, len(providerRegistry))
	for k := range providerRegistry {
		names = append(names, k)
	}
	return names
}
