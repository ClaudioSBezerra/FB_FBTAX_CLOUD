import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from '@/components/ui/dialog'
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { toast } from 'sonner'

// Administração de usuários do painel. O backend (handlers/usuarios.go) só mostra e altera
// usuários do mesmo ambiente de quem está logado; todo usuário criado aqui é administrador.

interface Usuario {
  id: string
  nome: string
  email: string
  perfil: string
  criado_em: string
  voce: boolean
}

async function lerErro(res: Response, padrao: string): Promise<string> {
  const texto = await res.text()
  try {
    const j = JSON.parse(texto)
    if (typeof j === 'string') return j
    if (j && typeof j.error === 'string') return j.error
  } catch { /* corpo não é JSON */ }
  return texto.trim() || padrao
}

function formatarData(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? '—' : d.toLocaleDateString('pt-BR')
}

export default function UsuariosPage() {
  const [usuarios, setUsuarios] = useState<Usuario[]>([])
  const [carregando, setCarregando] = useState(true)
  const [erro, setErro] = useState<string | null>(null)

  const [novoAberto, setNovoAberto] = useState(false)
  const [nome, setNome] = useState('')
  const [email, setEmail] = useState('')
  const [comSenha, setComSenha] = useState(false)
  const [senha, setSenha] = useState('')
  const [salvando, setSalvando] = useState(false)

  const [remover, setRemover] = useState<Usuario | null>(null)
  const [enviando, setEnviando] = useState<Record<string, boolean>>({})

  const carregar = async () => {
    setCarregando(true)
    try {
      const res = await fetch('/api/admin/usuarios')
      if (!res.ok) throw new Error(await lerErro(res, 'Erro ao carregar usuários'))
      setUsuarios(await res.json())
      setErro(null)
    } catch (e) {
      setErro(e instanceof Error ? e.message : 'Erro ao carregar usuários')
    } finally {
      setCarregando(false)
    }
  }

  useEffect(() => { carregar() }, [])

  const abrirNovo = () => {
    setNome(''); setEmail(''); setSenha(''); setComSenha(false)
    setNovoAberto(true)
  }

  const criar = async (e: React.FormEvent) => {
    e.preventDefault()
    if (comSenha && senha.length < 8) {
      toast.error('A senha inicial precisa ter pelo menos 8 caracteres')
      return
    }
    setSalvando(true)
    try {
      const res = await fetch('/api/admin/usuarios', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ nome, email, senha: comSenha ? senha : '' }),
      })
      if (!res.ok) throw new Error(await lerErro(res, 'Erro ao criar usuário'))
      const r = await res.json() as { convite_enviado?: boolean; aviso?: string }
      if (r.aviso) toast.warning(r.aviso)
      else if (r.convite_enviado) toast.success(`Usuário criado. Convite enviado para ${email.trim().toLowerCase()}`)
      else toast.success('Usuário criado. Passe a senha inicial para a pessoa por um canal seguro.')
      setNovoAberto(false)
      carregar()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Erro ao criar usuário')
    } finally {
      setSalvando(false)
    }
  }

  const reenviarConvite = async (u: Usuario) => {
    setEnviando(p => ({ ...p, [u.id]: true }))
    try {
      const res = await fetch(`/api/admin/usuarios/convite?id=${encodeURIComponent(u.id)}`, { method: 'POST' })
      if (!res.ok) throw new Error(await lerErro(res, 'Erro ao enviar convite'))
      toast.success(`Link enviado para ${u.email}`)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Erro ao enviar convite')
    } finally {
      setEnviando(p => ({ ...p, [u.id]: false }))
    }
  }

  const confirmarRemocao = async () => {
    if (!remover) return
    const alvo = remover
    setRemover(null)
    try {
      const res = await fetch(`/api/admin/usuarios?id=${encodeURIComponent(alvo.id)}`, { method: 'DELETE' })
      if (!res.ok) throw new Error(await lerErro(res, 'Erro ao remover usuário'))
      toast.success(`Acesso de ${alvo.nome || alvo.email} removido`)
      carregar()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Erro ao remover usuário')
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        {erro && (
          <Alert variant="destructive">
            <AlertDescription>{erro}</AlertDescription>
          </Alert>
        )}

        <Card>
          <CardHeader className="flex flex-row items-start justify-between gap-4">
            <div>
              <CardTitle>Usuários</CardTitle>
              <CardDescription className="mt-1">
                Quem tem acesso a este painel. Todo usuário é administrador e vê as mesmas telas que você.
              </CardDescription>
            </div>
            <Button onClick={abrirNovo}>Novo usuário</Button>
          </CardHeader>
          <CardContent>
            {carregando ? (
              <p className="text-sm text-muted-foreground py-6 text-center">Carregando…</p>
            ) : usuarios.length === 0 ? (
              <p className="text-sm text-muted-foreground py-6 text-center">Nenhum usuário.</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Nome</TableHead>
                    <TableHead>E-mail</TableHead>
                    <TableHead>Perfil</TableHead>
                    <TableHead>Desde</TableHead>
                    <TableHead className="text-right">Ações</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {usuarios.map(u => (
                    <TableRow key={u.id}>
                      <TableCell className="font-medium">
                        {u.nome || '—'} {u.voce && <Badge variant="secondary" className="ml-2">você</Badge>}
                      </TableCell>
                      <TableCell>{u.email}</TableCell>
                      <TableCell>{u.perfil === 'admin' ? 'Administrador' : u.perfil}</TableCell>
                      <TableCell>{formatarData(u.criado_em)}</TableCell>
                      <TableCell className="text-right space-x-2 whitespace-nowrap">
                        {!u.voce && (
                          <>
                            <Button variant="outline" size="sm" disabled={!!enviando[u.id]} onClick={() => reenviarConvite(u)}>
                              {enviando[u.id] ? 'Enviando…' : 'Enviar link de senha'}
                            </Button>
                            <Button variant="outline" size="sm" className="text-red-600 hover:text-red-700" onClick={() => setRemover(u)}>
                              Remover
                            </Button>
                          </>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Novo usuário */}
      <Dialog open={novoAberto} onOpenChange={setNovoAberto}>
        <DialogContent>
          <form onSubmit={criar} className="space-y-4">
            <DialogHeader>
              <DialogTitle>Novo usuário</DialogTitle>
              <DialogDescription>
                A pessoa recebe por e-mail um link para criar a própria senha (vale 72 horas).
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-2">
              <Label htmlFor="u-nome">Nome</Label>
              <Input id="u-nome" value={nome} onChange={e => setNome(e.target.value)} required autoFocus />
            </div>
            <div className="space-y-2">
              <Label htmlFor="u-email">E-mail</Label>
              <Input id="u-email" type="email" value={email} onChange={e => setEmail(e.target.value)} required autoCapitalize="none" autoCorrect="off" />
            </div>
            <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
              <input type="checkbox" checked={comSenha} onChange={e => setComSenha(e.target.checked)} />
              Definir uma senha inicial em vez de enviar o convite
            </label>
            {comSenha && (
              <div className="space-y-2">
                <Label htmlFor="u-senha">Senha inicial</Label>
                <Input id="u-senha" type="password" value={senha} onChange={e => setSenha(e.target.value)} minLength={8} autoComplete="new-password" />
                <p className="text-xs text-muted-foreground">Mínimo 8 caracteres. Passe para a pessoa por um canal seguro.</p>
              </div>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setNovoAberto(false)}>Cancelar</Button>
              <Button type="submit" disabled={salvando}>{salvando ? 'Salvando…' : 'Criar usuário'}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Confirmação de remoção */}
      <AlertDialog open={!!remover} onOpenChange={o => { if (!o) setRemover(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remover acesso?</AlertDialogTitle>
            <AlertDialogDescription>
              {remover?.nome || remover?.email} ({remover?.email}) deixa de entrar no painel. Para voltar, é preciso criar o usuário de novo.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancelar</AlertDialogCancel>
            <AlertDialogAction className="bg-red-600 hover:bg-red-700" onClick={confirmarRemocao}>Remover</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
