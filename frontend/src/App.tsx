import { useEffect, useState } from 'react'
import {
  ChevronLeft,
  ChevronRight,
  Database,
  Download,
  Loader2,
  LogOut,
  RefreshCcw,
  Search,
  ShieldCheck,
} from 'lucide-react'
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

  if (checkingSession) {
    return <LoadingScreen />
  }

  if (!user) {
    return <LoginScreen onLoggedIn={setUser} />
  }

  return (
    <Dashboard
      user={user}
      onLogout={() => {
        clearToken()
        setUser(null)
      }}
    />
  )
}

function LoadingScreen() {
  return (
    <main className="grid min-h-screen place-items-center px-4">
      <div className="panel flex items-center gap-3 px-5 py-4 text-sm text-muted">
        <Loader2 className="size-4 animate-spin text-accent" />
        Carregando sessão
      </div>
    </main>
  )
}

function LoginScreen({ onLoggedIn }: { onLoggedIn: (user: User) => void }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)
    setError('')

    try {
      const { token } = await api.login(email.trim(), password)
      setToken(token)
      const currentUser = await api.me()
      onLoggedIn(currentUser)
    } catch (err) {
      clearToken()
      setError(err instanceof ApiError && err.status === 401 ? 'Credenciais inválidas' : errorMessage(err, 'Não foi possível entrar'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="grid min-h-screen place-items-center px-4 py-10">
      <section className="login-shell">
        <div className="login-brand">
          <div className="brand-mark">
            <ShieldCheck className="size-5" />
          </div>
          <div>
            <p className="eyebrow">Acesso interno</p>
            <h1>Painel UY3</h1>
          </div>
        </div>

        <form className="space-y-4" onSubmit={handleSubmit}>
          <div>
            <label className="field-label" htmlFor="email">
              Email
            </label>
            <input
              id="email"
              className="field"
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </div>

          <div>
            <label className="field-label" htmlFor="password">
              Senha
            </label>
            <input
              id="password"
              className="field"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </div>

          {error && <div className="alert">{error}</div>}

          <button className="button button-primary w-full" type="submit" disabled={loading}>
            {loading ? <Loader2 className="size-4 animate-spin" /> : <ShieldCheck className="size-4" />}
            Entrar
          </button>
        </form>
      </section>
    </main>
  )
}

function Dashboard({ user, onLogout }: { user: User; onLogout: () => void }) {
  const [period, setPeriod] = useState('7d')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [sort, setSort] = useState<'received_at' | 'id'>('received_at')
  const [direction, setDirection] = useState<'asc' | 'desc'>('desc')
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(20)
  const [data, setData] = useState<LeadsResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [error, setError] = useState('')

  async function fetchLeads(targetPage = page) {
    setLoading(true)
    setError('')

    try {
      const response = await api.leads(buildFilters(targetPage))
      setData(response)
      setPage(response.current_page)
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        onLogout()
        return
      }
      setError(errorMessage(err, 'Não foi possível carregar os leads'))
    } finally {
      setLoading(false)
    }
  }

  async function handleExport() {
    setExporting(true)
    setError('')

    try {
      await downloadLeadsExport(buildFilters(1))
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        onLogout()
        return
      }
      setError(errorMessage(err, 'Não foi possível exportar o CSV'))
    } finally {
      setExporting(false)
    }
  }

  function buildFilters(targetPage: number): LeadFilters {
    return {
      page: targetPage,
      per_page: perPage,
      sort,
      direction,
      ...(from || to ? { from, to } : { period }),
    }
  }

  useEffect(() => {
    // A carga inicial sincroniza a tela com a API assim que o dashboard monta.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void fetchLeads(1)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const leads = data?.items ?? []
  const visibleValue = leads.reduce((total, lead) => total + numberValue(lead.valor_liberado), 0)
  const eligibleCount = leads.filter((lead) => boolLabel(lead.elegivel_emprestimo) === 'Sim').length

  return (
    <div className="min-h-screen">
      <header className="topbar">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-3">
            <div className="brand-mark">
              <Database className="size-5" />
            </div>
            <div>
              <h1 className="text-base font-semibold text-strong">Painel UY3</h1>
              <p className="hidden text-xs text-muted sm:block">Leads recebidos via webhook</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <span className="hidden text-sm text-muted md:inline">{user.email}</span>
            <button className="button button-muted" type="button" onClick={onLogout}>
              <LogOut className="size-4" />
              Sair
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:py-8">
        <section className="mb-6 grid gap-3 md:grid-cols-3">
          <Metric label="Total filtrado" value={formatInteger(data?.total ?? 0)} />
          <Metric label="Elegíveis na página" value={formatInteger(eligibleCount)} />
          <Metric label="Valor na página" value={formatBRL(visibleValue)} />
        </section>

        <section className="panel mb-5 p-4">
          <div className="grid gap-4 lg:grid-cols-[1fr_1fr_1fr_0.85fr_0.85fr_auto_auto]">
            <div>
              <label className="field-label" htmlFor="period">
                Periodo
              </label>
              <select
                id="period"
                className="field"
                value={period}
                onChange={(event) => setPeriod(event.target.value)}
                disabled={Boolean(from || to)}
              >
                {periods.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="field-label" htmlFor="from">
                De
              </label>
              <input id="from" className="field" type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
            </div>

            <div>
              <label className="field-label" htmlFor="to">
                Até
              </label>
              <input id="to" className="field" type="date" value={to} onChange={(event) => setTo(event.target.value)} />
            </div>

            <div>
              <label className="field-label" htmlFor="sort">
                Ordenar por
              </label>
              <select id="sort" className="field" value={sort} onChange={(event) => setSort(event.target.value as 'received_at' | 'id')}>
                <option value="received_at">Recebimento</option>
                <option value="id">ID</option>
              </select>
            </div>

            <div>
              <label className="field-label" htmlFor="direction">
                Direção
              </label>
              <select id="direction" className="field" value={direction} onChange={(event) => setDirection(event.target.value as 'asc' | 'desc')}>
                <option value="desc">Desc</option>
                <option value="asc">Asc</option>
              </select>
            </div>

            <div className="flex items-end">
              <button className="button button-primary w-full" type="button" onClick={() => void fetchLeads(1)} disabled={loading}>
                {loading ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
                Buscar
              </button>
            </div>

            <div className="flex items-end">
              <button className="button button-muted w-full" type="button" onClick={() => void handleExport()} disabled={exporting}>
                {exporting ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                CSV
              </button>
            </div>
          </div>

          {(from || to) && (
            <div className="mt-3 flex items-center justify-between gap-3 rounded-md border border-border bg-soft px-3 py-2 text-xs text-muted">
              <span>Filtro por datas ativo. O período fica pausado enquanto houver data preenchida.</span>
              <button
                className="inline-flex items-center gap-1 font-medium text-accent"
                type="button"
                onClick={() => {
                  setFrom('')
                  setTo('')
                }}
              >
                <RefreshCcw className="size-3" />
                Limpar
              </button>
            </div>
          )}
        </section>

        {error && <div className="alert mb-5">{error}</div>}

        <section className="panel overflow-hidden">
          <div className="overflow-x-auto">
            <table className="data-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Recebido em</th>
                  <th>CPF</th>
                  <th>Nome</th>
                  <th>Status</th>
                  <th>Elegivel</th>
                  <th className="text-right">Valor liberado</th>
                  <th className="text-right">Margem</th>
                  <th className="text-right">Parcelas</th>
                </tr>
              </thead>
              <tbody>
                {loading && (
                  <tr>
                    <td className="table-state" colSpan={9}>
                      <Loader2 className="mr-2 inline size-4 animate-spin text-accent" />
                      Carregando leads
                    </td>
                  </tr>
                )}

                {!loading && leads.length === 0 && (
                  <tr>
                    <td className="table-state" colSpan={9}>
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

          <div className="flex flex-col gap-3 border-t border-border bg-soft px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3 text-xs text-muted">
              <span>
                Página {data?.current_page ?? 1} de {Math.max(data?.total_pages ?? 1, 1)}
              </span>
              <select
                className="field h-9 w-24"
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
              <button className="button button-muted" type="button" disabled={!data?.has_previous || loading} onClick={() => void fetchLeads(page - 1)}>
                <ChevronLeft className="size-4" />
                Anterior
              </button>
              <button className="button button-muted" type="button" disabled={!data?.has_next || loading} onClick={() => void fetchLeads(page + 1)}>
                Próxima
                <ChevronRight className="size-4" />
              </button>
            </div>
          </div>
        </section>
      </main>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="panel p-4">
      <p className="text-xs font-medium uppercase text-muted">{label}</p>
      <p className="mt-2 text-2xl font-semibold text-strong">{value}</p>
    </div>
  )
}

function LeadRow({ lead }: { lead: Lead }) {
  return (
    <tr>
      <td className="font-mono text-xs text-muted">#{lead.id}</td>
      <td className="whitespace-nowrap">{formatDateTime(lead.received_at)}</td>
      <td className="font-mono text-xs">{formatCPF(lead.cpf)}</td>
      <td className="min-w-52">{lead.nome_trabalhador || '-'}</td>
      <td>
        <span className="pill">{lead.status || '-'}</span>
      </td>
      <td>
        <span className={boolLabel(lead.elegivel_emprestimo) === 'Sim' ? 'pill pill-ok' : 'pill pill-muted'}>{boolLabel(lead.elegivel_emprestimo)}</span>
      </td>
      <td className="whitespace-nowrap text-right font-medium">{formatBRL(lead.valor_liberado)}</td>
      <td className="whitespace-nowrap text-right">{formatBRL(lead.margem_disponivel)}</td>
      <td className="text-right">{lead.numero_parcelas || '-'}</td>
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
