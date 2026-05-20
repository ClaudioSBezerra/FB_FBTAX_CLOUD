export interface ViaCEPResult {
  logradouro: string
  bairro: string
  localidade: string
  uf: string
}

export async function buscarCEP(cep: string): Promise<ViaCEPResult | null> {
  const digits = cep.replace(/\D/g, '')
  if (digits.length !== 8) return null
  try {
    const res = await fetch(`https://viacep.com.br/ws/${digits}/json/`)
    if (!res.ok) return null
    const data = await res.json()
    if (data.erro) return null
    return data as ViaCEPResult
  } catch {
    return null
  }
}
