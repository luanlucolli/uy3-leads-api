import { useEffect, useState } from 'react'
import {
  ChevronLeft,
  ChevronRight,
  Database,
  Download,
  Filter,
  Loader2,
  LogOut,
  RefreshCcw,
  Search,
  ShieldCheck,
} from 'lucide-react'
import { Toaster, toast } from 'react-hot-toast'
import {
  ApiError,
  api,
  clearToken,
  downloadLeadsExport,
  getToken,
  setToken,
  type Lead,
  type LeadFilters,
  type LeadsResponse,
  type User,
} from './lib/api'

const periods = [
  { value: 'all', label: 'Todos' },
  { value: '24h', label: '24h' },
  { value: '7d', label: '7 dias' },
  { value: '30d', label: '30 dias' },
  { value: '90d', label: '90 dias' },
  { value: 'custom', label: 'Personalizado' },
]

function App() {
  const [user, setUser] = useState<User | null>(null)
  const [checkingSession, setCheckingSession] = useState(true)

  useEffect(() => {
    async function loadSession() {
      if (!getToken()) {
        setCheckingSession(false)
        return
      }

      try {
        const currentUser = await api.me()
        setUser(currentUser)
      } catch {
        clearToken()
      } finally {
        setCheckingSession(false)
      }
    }

    void loadSession()
  }, [])

  return (
    <>
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 3500,
          style: {
            background: '#151922',
            color: '#eef2f6',
            border: '1px solid #2a303c',
            boxShadow: '0 18px 48px rgb(0 0 0 / 28%)',
          },
          success: {
            iconTheme: {
              primary: '#2dd4bf',
              secondary: '#151922',
            },
          },
          error: {
            iconTheme: {
              primary: '#ef4444',
              secondary: '#151922',
            },
          },
        }}
      />

      {checkingSession ? (
        <LoadingScreen />
      ) : user ? (
        <Dashboard
          user={user}
          onLogout={() => {
            clearToken()
            setUser(null)
          }}
        />
      ) : (
        <LoginScreen onLoggedIn={setUser} />
      )}
    </>
  )
}

function LoadingScreen() {
  return (
    <main className="grid min-h-screen place-items-center px-4">
      <div className="panel flex items-center gap-3 px-5 py-4 text-base text-muted">
        <Loader2 className="size-5 animate-spin text-accent" />
        Carregando sessão
      </div>
    </main>
  )
}

function LoginScreen({ onLoggedIn }: { onLoggedIn: (user: User) => void }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)

    try {
      const { token } = await api.login(email.trim(), password)
      setToken(token)
      const currentUser = await api.me()
      onLoggedIn(currentUser)
    } catch (err) {
      clearToken()
      toast.error(err instanceof ApiError && err.status === 401 ? 'Credenciais inválidas' : errorMessage(err, 'Não foi possível entrar'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="grid min-h-screen place-items-center px-4 py-10">
      <section className="login-shell">
        <div className="login-brand mb-6 flex flex-col items-center text-center">
          <div className="brand-mark mb-4 rounded-full bg-soft p-3">
            <ShieldCheck className="size-8 text-strong" />
          </div>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-muted">Acesso interno</p>
            <h1 className="mt-1 text-2xl font-bold text-strong">Painel Leads UY3</h1>
          </div>
        </div>

        <form className="space-y-5" onSubmit={handleSubmit}>
          <div>
            <label className="field-label text-sm font-medium" htmlFor="email">
              Email
            </label>
            <input
              id="email"
              className="field mt-1 w-full text-base"
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </div>

          <div>
            <label className="field-label text-sm font-medium" htmlFor="password">
              Senha
            </label>
            <input
              id="password"
              className="field mt-1 w-full text-base"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </div>

          <button className="button button-primary w-full text-base font-medium" type="submit" disabled={loading}>
            {loading ? <Loader2 className="size-5 animate-spin" /> : <ShieldCheck className="size-5" />}
            Entrar
          </button>
        </form>
      </section>
    </main>
  )
}

function Dashboard({ user, onLogout }: { user: User; onLogout: () => void }) {
  const [interval, setInterval] = useState('7d')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [direction, setDirection] = useState<'asc' | 'desc'>('desc')
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(20)
  const [data, setData] = useState<LeadsResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [logoutDialogOpen, setLogoutDialogOpen] = useState(false)

  async function fetchLeads(targetPage = page) {
    setLoading(true)

    try {
      const response = await api.leads(buildFilters(targetPage))
      setData(response)
      setPage(response.current_page)
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        onLogout()
        return
      }
      toast.error(errorMessage(err, 'Não foi possível carregar os leads'))
    } finally {
      setLoading(false)
    }
  }

  async function handleExport() {
    setExporting(true)

    try {
      await downloadLeadsExport(buildFilters(1))
      toast.success('Download iniciado!')
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        onLogout()
        return
      }
      toast.error(errorMessage(err, 'Não foi possível exportar o CSV'))
    } finally {
      setExporting(false)
    }
  }

  function buildFilters(targetPage: number): LeadFilters {
    return {
      page: targetPage,
      per_page: perPage,
      sort: 'received_at',
      direction,
      ...(interval === 'custom' ? { from, to } : { period: interval }),
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void fetchLeads(1)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const leads = data?.items ?? []

  return (
    <div className="min-h-screen text-base">
      <header className="topbar">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-3">
            <div className="brand-mark">
              <Database className="size-6" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-strong">Painel UY3</h1>
              <p className="hidden text-sm text-muted sm:block">Leads recebidos via webhook</p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <span className="hidden text-sm font-medium text-muted md:inline">{user.email}</span>
            <button className="button button-muted text-sm font-medium" type="button" onClick={() => setLogoutDialogOpen(true)}>
              <LogOut className="size-4" />
              Sair
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:py-8">
        <section className="panel mb-6">
          <div className="flex items-center justify-between border-b border-border bg-soft/30 px-5 py-4">
            <h2 className="flex items-center gap-2 text-base font-semibold text-strong">
              <Filter className="size-5 text-muted" />
              Filtros e Exportação
            </h2>
            <div className="flex items-center gap-2 text-sm font-medium text-muted">
              <span>Total filtrado:</span>
              <span className="rounded-md bg-border/50 px-2 py-0.5 text-strong">
                {formatInteger(data?.total ?? 0)}
              </span>
            </div>
          </div>
          
          <div className="p-5">
            <div className="flex flex-col gap-5 sm:flex-row sm:flex-wrap sm:items-end">
              <div className="min-w-[200px] flex-1">
                <label className="field-label mb-1 block text-sm font-medium" htmlFor="interval">
                  Intervalo
                </label>
                <select
                  id="interval"
                  className="field w-full text-sm"
                  value={interval}
                  onChange={(event) => {
                    const nextInterval = event.target.value
                    setInterval(nextInterval)
                    if (nextInterval !== 'custom') {
                      setFrom('')
                      setTo('')
                    }
                  }}
                >
                  {periods.map((item) => (
                    <option key={item.value} value={item.value}>
                      {item.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="min-w-[200px] flex-1">
                <label className="field-label mb-1 block text-sm font-medium" htmlFor="order">
                  Ordenação
                </label>
                <select id="order" className="field w-full text-sm" value={direction} onChange={(event) => setDirection(event.target.value as 'asc' | 'desc')}>
                  <option value="desc">Mais recente</option>
                  <option value="asc">Mais antigo</option>
                </select>
              </div>

              {interval === 'custom' && (
                <div className="flex w-full flex-col gap-4 sm:w-auto sm:flex-1 sm:flex-row sm:items-end md:min-w-[340px]">
                  <div className="flex-1">
                    <label className="field-label mb-1 block text-sm font-medium" htmlFor="from">
                      De
                    </label>
                    <input id="from" className="field w-full text-sm" type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
                  </div>

                  <div className="flex-1">
                    <label className="field-label mb-1 block text-sm font-medium" htmlFor="to">
                      Até
                    </label>
                    <input id="to" className="field w-full text-sm" type="date" value={to} onChange={(event) => setTo(event.target.value)} />
                  </div>

                  <button
                    className="button button-muted mt-2 text-sm sm:mt-0"
                    type="button"
                    onClick={() => {
                      setFrom('')
                      setTo('')
                    }}
                    title="Limpar datas"
                  >
                    <RefreshCcw className="size-4" />
                    <span className="sm:hidden">Limpar datas</span>
                  </button>
                </div>
              )}
            </div>

            <div className="mt-6 flex flex-col-reverse items-center justify-end gap-3 border-t border-border pt-5 sm:flex-row">
              <button className="button button-muted w-full text-sm font-medium sm:w-auto" type="button" onClick={() => void handleExport()} disabled={exporting}>
                {exporting ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                Exportar CSV
              </button>
              
              <button className="button button-primary w-full text-sm font-medium sm:w-auto" type="button" onClick={() => void fetchLeads(1)} disabled={loading}>
                {loading ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
                Buscar Leads
              </button>
            </div>
          </div>
        </section>

        <section className="panel overflow-hidden">
          <div className="overflow-x-auto">
            <table className="data-table w-full text-sm">
              <thead className="text-left font-semibold text-muted">
                <tr>
                  <th className="py-3 pr-4">ID</th>
                  <th className="px-4 py-3">Recebido em</th>
                  <th className="px-4 py-3">CPF</th>
                  <th className="px-4 py-3">Nome</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Elegível</th>
                  <th className="px-4 py-3 text-right">Valor liberado</th>
                  <th className="px-4 py-3 text-right">Margem</th>
                  <th className="py-3 pl-4 text-right">Parcelas</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {loading && (
                  <tr>
                    <td className="table-state py-8 text-center text-muted" colSpan={9}>
                      <Loader2 className="mr-2 inline size-5 animate-spin text-accent" />
                      Carregando leads
                    </td>
                  </tr>
                )}

                {!loading && leads.length === 0 && (
                  <tr>
                    <td className="table-state py-8 text-center text-muted" colSpan={9}>
                      Nenhum lead encontrado para os filtros selecionados.
                    </td>
                  </tr>
                )}

                {!loading &&
                  leads.map((lead) => (
                    <LeadRow key={lead.id} lead={lead} />
                  ))}
              </tbody>
            </table>
          </div>

          <div className="flex flex-col gap-4 border-t border-border bg-soft px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3 text-sm font-medium text-muted">
              <span>
                Página {data?.current_page ?? 1} de {Math.max(data?.total_pages ?? 1, 1)}
              </span>
              <select
                className="field h-9 w-20 text-sm"
                value={perPage}
                onChange={(event) => {
                  setPerPage(Number(event.target.value))
                  setPage(1)
                }}
              >
                <option value={20}>20</option>
                <option value={50}>50</option>
                <option value={100}>100</option>
              </select>
            </div>

            <div className="flex items-center gap-2">
              <button className="button button-muted text-sm font-medium" type="button" disabled={!data?.has_previous || loading} onClick={() => void fetchLeads(page - 1)}>
                <ChevronLeft className="size-4" />
                Anterior
              </button>
              <button className="button button-muted text-sm font-medium" type="button" disabled={!data?.has_next || loading} onClick={() => void fetchLeads(page + 1)}>
                Próxima
                <ChevronRight className="size-4" />
              </button>
            </div>
          </div>
        </section>
      </main>

      {logoutDialogOpen && (
        <div
          className="dialog-backdrop"
          role="presentation"
          onClick={() => setLogoutDialogOpen(false)}
        >
          <div
            className="dialog-panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="logout-dialog-title"
            aria-describedby="logout-dialog-description"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="dialog-content">
              <h2 id="logout-dialog-title" className="dialog-title">
                Confirmar Saída
              </h2>
              <p id="logout-dialog-description" className="dialog-description">
                Tem certeza que deseja encerrar sua sessão?
              </p>
            </div>

            <div className="dialog-actions">
              <button
                className="button button-muted"
                type="button"
                onClick={() => setLogoutDialogOpen(false)}
              >
                Cancelar
              </button>
              <button
                className="button button-danger"
                type="button"
                onClick={() => {
                  setLogoutDialogOpen(false)
                  onLogout()
                }}
              >
                Sair
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function LeadRow({ lead }: { lead: Lead }) {
  return (
    <tr className="hover:bg-soft/50">
      <td className="py-3 pr-4 font-mono text-xs text-muted">#{lead.id}</td>
      <td className="whitespace-nowrap px-4 py-3">{formatDateTime(lead.received_at)}</td>
      <td className="px-4 py-3 font-mono text-sm">{formatCPF(lead.cpf)}</td>
      <td className="min-w-52 px-4 py-3 font-medium">{lead.nome_trabalhador || '-'}</td>
      <td className="px-4 py-3">
        <span className="pill text-xs font-medium">{lead.status || '-'}</span>
      </td>
      <td className="px-4 py-3">
        <span className={boolLabel(lead.elegivel_emprestimo) === 'Sim' ? 'pill pill-ok text-xs font-medium' : 'pill pill-muted text-xs font-medium'}>{boolLabel(lead.elegivel_emprestimo)}</span>
      </td>
      <td className="whitespace-nowrap px-4 py-3 text-right font-medium">{formatBRL(lead.valor_liberado)}</td>
      <td className="whitespace-nowrap px-4 py-3 text-right">{formatBRL(lead.margem_disponivel)}</td>
      <td className="py-3 pl-4 text-right">{lead.numero_parcelas || '-'}</td>
    </tr>
  )
}

function errorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) {
    return error.message
  }
  return fallback
}

function numberValue(value: number | string | null | undefined) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

function formatBRL(value: number | string | null | undefined) {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
  }).format(numberValue(value))
}

function formatInteger(value: number) {
  return new Intl.NumberFormat('pt-BR').format(value)
}

function formatDateTime(value: string) {
  if (!value) {
    return '-'
  }

  const date = new Date(value.replace(' ', 'T'))
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return date.toLocaleString('pt-BR')
}

function formatCPF(value: string) {
  const digits = value.replace(/\D/g, '')
  if (digits.length !== 11) {
    return value || '-'
  }
  return `${digits.slice(0, 3)}.${digits.slice(3, 6)}.${digits.slice(6, 9)}-${digits.slice(9)}`
}

function boolLabel(value: string) {
  const normalized = String(value ?? '').trim().toLowerCase()

  if (['true', '1', 'sim', 's', 'yes'].includes(normalized)) {
    return 'Sim'
  }
  if (['false', '0', 'nao', 'não', 'n', 'no'].includes(normalized)) {
    return 'Não'
  }
  return value || '-'
}

export default App
