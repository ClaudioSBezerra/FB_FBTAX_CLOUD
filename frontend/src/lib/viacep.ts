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
    const res = await fetch(`/api/cep?cep=${digits}`)
    if (!res.ok) return null
    return res.json() as Promise<ViaCEPResult>
  } catch {
    return null
  }
}
