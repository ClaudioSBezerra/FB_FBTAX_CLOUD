import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { useNavigate, Link } from "react-router-dom";
import { useAuth } from "@/contexts/AuthContext";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { ThemeToggle } from "@/components/ThemeToggle";
import { AlertCircle, Users, FileText, KeyRound, LayoutDashboard } from "lucide-react";

const FEATURES = [
  { icon: Users,           text: "Gestão de clientes e contratos"          },
  { icon: KeyRound,        text: "Tokens de licença com ciclo de vida"     },
  { icon: FileText,        text: "Portal de acesso para clientes"          },
  { icon: LayoutDashboard, text: "Dashboard financeiro em tempo real"      },
];

const Login = () => {
  const [email, setEmail]       = useState("");
  const [password, setPassword] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg]   = useState<string | null>(null);
  const navigate = useNavigate();
  const { login } = useAuth();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setErrorMsg(null);

    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(typeof data === "string" ? data : "Credenciais inválidas");
      }

      login(data);
      toast.success("Login realizado com sucesso!");
      navigate("/admin/financeiro");
    } catch (error: unknown) {
      const msg = error instanceof Error ? error.message : "Erro desconhecido";
      setErrorMsg(msg);
      toast.error(msg);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex">

      {/* ── Painel esquerdo — identidade Fortes Bezerra ──────────────────── */}
      <div className="hidden lg:flex lg:w-2/5 flex-col justify-between p-10 relative overflow-hidden bg-[#0f172a] dark:bg-gradient-to-br dark:from-[#0b1620] dark:to-[#0a1a2b] dark:border-r dark:border-teal-500/15">
        {/* Círculos decorativos — no dark viram orbes teal/cyan */}
        <div
          className="absolute -top-24 -right-24 w-96 h-96 rounded-full pointer-events-none bg-[radial-gradient(circle,#1e3a5f_0%,#0f172a_100%)] dark:bg-none"
          aria-hidden="true"
        />
        <div
          className="absolute -bottom-20 -left-20 w-64 h-64 rounded-full pointer-events-none bg-[radial-gradient(circle,#162032_0%,#0f172a_100%)] dark:bg-none"
          aria-hidden="true"
        />
        <div className="ambient-light -top-16 right-[10%]" aria-hidden="true" />
        <div className="ambient-light-2 -bottom-32 -left-16" aria-hidden="true" />

        {/* ── Topo: logotipo do produto ── */}
        <div className="relative z-10">
          <div className="flex items-center gap-3 mb-12">
            <div className="w-10 h-10 rounded-xl flex items-center justify-center font-bold text-white dark:text-[#0b1620] text-sm tracking-wide select-none bg-[#2563eb] dark:bg-gradient-to-br dark:from-teal-500 dark:to-cyan-400 glow-teal">
              FB
            </div>
            <span className="text-white text-xl font-bold tracking-tight text-glow">Fortes Bezerra</span>
          </div>

          {/* Badge */}
          <span className="inline-block px-4 py-1.5 rounded-full text-sm uppercase tracking-widest font-semibold border bg-blue-600/15 text-blue-300 border-blue-600/30 dark:bg-teal-500/15 dark:text-teal-300 dark:border-teal-400/40 backdrop-glow">
            Módulo Financeiro
          </span>

          {/* Título */}
          <h1 className="text-white text-4xl font-bold leading-tight mt-5 text-glow">
            Contratos e licenças
            <br />
            <span className="text-blue-400 dark:text-teal-300">sob controle.</span>
          </h1>

          {/* Subtítulo */}
          <p className="mt-5 text-base leading-relaxed text-slate-400">
            Gerencie clientes, contratos e tokens de acesso aos produtos FB —
            com portal de autoatendimento e rastreabilidade completa.
          </p>
        </div>

        {/* ── Rodapé: features ── */}
        <div className="relative z-10 space-y-4">
          <ul className="space-y-3">
            {FEATURES.map(({ icon: Icon, text }) => (
              <li key={text} className="flex items-center gap-3 text-sm text-slate-300">
                <div className="w-6 h-6 rounded-md flex items-center justify-center shrink-0 bg-blue-600/20 dark:bg-teal-500/20 dark:border dark:border-teal-400/30">
                  <Icon className="w-3.5 h-3.5 text-blue-400 dark:text-teal-300" />
                </div>
                {text}
              </li>
            ))}
          </ul>

          <p className="text-xs pt-2 text-slate-600 dark:text-slate-500">
            © {new Date().getFullYear()} Fortes Bezerra Tecnologia · FBTax Cloud
          </p>
        </div>
      </div>

      {/* ── Painel direito — formulário de login ─────────────────────────── */}
      <div className="relative flex-1 flex items-center justify-center bg-background px-4">
        <ThemeToggle className="absolute top-4 right-4" />

        <div className="w-full max-w-[420px]">

          {/* Logo mobile (só aparece em telas pequenas) */}
          <div className="flex lg:hidden items-center justify-center gap-2 mb-8">
            <div className="w-8 h-8 rounded-lg flex items-center justify-center font-bold text-white dark:text-[#0b1620] text-xs tracking-wide select-none bg-[#2563eb] dark:bg-gradient-to-br dark:from-teal-500 dark:to-cyan-400">
              FB
            </div>
            <span className="text-lg font-bold text-foreground">Fortes Bezerra</span>
          </div>

          <Card className="w-full shadow-md border-0 dark:border dark:border-border glow-card">
            <CardHeader className="flex flex-col items-center gap-1 space-y-0 pt-7 pb-4">
              <CardTitle className="text-base font-semibold">Acesse sua conta</CardTitle>
              <CardDescription className="text-xs">
                Entre com suas credenciais para continuar
              </CardDescription>
            </CardHeader>

            <CardContent>
              {errorMsg && (
                <Alert variant="destructive" className="mb-4">
                  <AlertCircle className="h-4 w-4" />
                  <AlertTitle>Erro</AlertTitle>
                  <AlertDescription>{errorMsg}</AlertDescription>
                </Alert>
              )}

              <form onSubmit={handleLogin} className="space-y-4">
                <div className="space-y-1.5">
                  <Label htmlFor="email" className="text-sm">E-mail</Label>
                  <Input
                    id="email"
                    type="email"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="seu@email.com"
                    className="text-sm"
                  />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="password" className="text-sm">Senha</Label>
                  <Input
                    id="password"
                    type="password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="text-sm"
                  />
                </div>

                <div className="flex justify-end">
                  <Link to="/forgot-password" className="text-xs text-primary hover:underline">
                    Esqueci minha senha
                  </Link>
                </div>

                <Button type="submit" className="w-full text-sm button-hover" disabled={isLoading}>
                  {isLoading ? "Entrando..." : "Entrar"}
                </Button>

                <p className="text-center text-xs text-muted-foreground mt-1">
                  Não tem uma conta?{" "}
                  <Link to="/register" className="text-primary hover:underline">
                    Crie grátis
                  </Link>
                </p>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>

    </div>
  );
};

export default Login;
