import { useEffect, useState } from 'react'
import {
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
  type LeadFilters,
  type LeadsSummaryResponse,
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

type DashboardInterval = (typeof periods)[number]['value']

type DashboardFilters = {
  interval: DashboardInterval
  from: string
  to: string
}

const dashboardFiltersStorageKey = 'uy3_dashboard_filters'
const periodValues = new Set<DashboardInterval>(periods.map((item) => item.value))
const defaultDashboardFilters: DashboardFilters = {
  interval: '7d',
  from: '',
  to: '',
}

const panelClass =
  'border border-border bg-[rgb(21_25_34_/_88%)] shadow-[inset_0_1px_0_rgb(255_255_255_/_4%),0_18px_48px_rgb(0_0_0_/_28%)]'

const brandMarkClass =
  'inline-grid h-[2.4rem] w-[2.4rem] shrink-0 place-items-center rounded-[8px] text-[#1a1008] bg-[linear-gradient(135deg,var(--accent),var(--accent-2))] shadow-[0_12px_30px_rgb(243_111_33_/_28%)]'

const fieldLabelClass =
  'mb-[0.45rem] block text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted'

const fieldClass =
  'h-10 w-full rounded-[6px] border border-border bg-[#10141b] px-3 text-text outline-none transition-[border-color,box-shadow,background] duration-150 ease-[ease] focus:border-accent focus:shadow-[0_0_0_3px_rgb(243_111_33_/_18%)]'

const buttonBaseClass =
  'inline-flex min-h-10 items-center justify-center gap-2 rounded-[6px] border border-transparent px-[0.85rem] text-[0.9rem] font-bold transition-[transform,border-color,background] duration-150 ease-[ease] active:translate-y-px disabled:cursor-not-allowed disabled:opacity-[0.55]'

const buttonPrimaryClass =
  `${buttonBaseClass} bg-[linear-gradient(135deg,var(--accent),var(--accent-2))] text-[#1a1008] shadow-[0_14px_34px_rgb(243_111_33_/_22%)]`

const buttonMutedClass =
  `${buttonBaseClass} border-border bg-surface-soft text-text hover:border-[#3b4352] hover:bg-[#222837]`

const buttonDangerClass =
  `${buttonBaseClass} border-[rgb(239_68_68_/_38%)] bg-[linear-gradient(180deg,rgb(239_68_68_/_18%),rgb(239_68_68_/_12%))] text-[#fff4f4] hover:border-[rgb(239_68_68_/_55%)] hover:bg-[linear-gradient(180deg,rgb(239_68_68_/_26%),rgb(239_68_68_/_18%))]`

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
      <div className={`${panelClass} flex items-center gap-3 rounded-[8px] px-5 py-4 text-base text-muted`}>
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
      <section className={`${panelClass} w-full max-w-[430px] rounded-[10px] p-8`}>
        <div className="mb-6 flex flex-col items-center gap-[0.85rem] text-center">
          <div className={`${brandMarkClass} mb-4 rounded-full bg-surface-soft p-3`}>
            <ShieldCheck className="size-8 text-text" />
          </div>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-muted">Acesso interno</p>
            <h1 className="mt-1 text-2xl font-bold text-text">Painel Leads UY3</h1>
          </div>
        </div>

        <form className="space-y-5" onSubmit={handleSubmit}>
          <div>
            <label className={`${fieldLabelClass} text-sm font-medium`} htmlFor="email">
              Email
            </label>
            <input
              id="email"
              className={`${fieldClass} mt-1 text-base`}
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </div>

          <div>
            <label className={`${fieldLabelClass} text-sm font-medium`} htmlFor="password">
              Senha
            </label>
            <input
              id="password"
              className={`${fieldClass} mt-1 text-base`}
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </div>

          <button className={`${buttonPrimaryClass} w-full text-base font-medium`} type="submit" disabled={loading}>
            {loading ? <Loader2 className="size-5 animate-spin" /> : <ShieldCheck className="size-5" />}
            Entrar
          </button>
        </form>
      </section>
    </main>
  )
}

function Dashboard({ user, onLogout }: { user: User; onLogout: () => void }) {
  const [draftFilters, setDraftFilters] = useState<DashboardFilters>(() => loadStoredDashboardFilters())
  const [appliedFilters, setAppliedFilters] = useState<DashboardFilters>(() => loadStoredDashboardFilters())
  const [data, setData] = useState<LeadsSummaryResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [logoutDialogOpen, setLogoutDialogOpen] = useState(false)

  async function fetchSummary(nextFilters = appliedFilters) {
    setLoading(true)

    try {
      const normalizedFilters = normalizeDashboardFilters(nextFilters)
      const response = await api.leads(buildFilters(normalizedFilters))
      setData(response)
      setAppliedFilters(normalizedFilters)
      saveDashboardFilters(normalizedFilters)
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
      await downloadLeadsExport(buildFilters(appliedFilters))
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

  function buildFilters(filters: DashboardFilters): LeadFilters {
    return {
      ...(filters.interval === 'custom'
        ? filters.from || filters.to
          ? { from: filters.from, to: filters.to }
          : { period: 'all' }
        : { period: filters.interval }),
    }
  }

  useEffect(() => {
    void fetchSummary(appliedFilters)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const total = data?.total ?? 0
  const hasLeads = total > 0

  return (
    <div className="min-h-screen text-base">
      <header className="sticky top-0 z-20 border-b border-border bg-[rgb(15_17_21_/_84%)] backdrop-blur-[16px]">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-3">
            <div className={brandMarkClass}>
              <Database className="size-6" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-text">Painel UY3</h1>
              <p className="hidden text-sm text-muted sm:block">Gestão de Leads</p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <span className="hidden text-sm font-medium text-muted md:inline">{user.email}</span>
            <button className={`${buttonMutedClass} h-8 px-3 text-sm font-medium`} type="button" onClick={() => setLogoutDialogOpen(true)}>
              <LogOut className="size-4" />
              Sair
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:py-8">

        {/* ÁREA DE BUSCA */}
        <section className={`${panelClass} mb-8 rounded-[8px] p-5`}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-bold text-muted uppercase tracking-widest">
            <Filter className="size-4" />
            Filtros de Busca
          </h2>

          <div className="flex flex-col gap-4 sm:flex-row sm:flex-wrap sm:items-end">
            <div className="min-w-[160px] flex-1">
              <label className={fieldLabelClass} htmlFor="interval">
                Intervalo
              </label>
              <select
                id="interval"
                className={`${fieldClass} text-sm appearance-none bg-[url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20fill%3D%22none%22%20viewBox%3D%220%200%2020%2020%22%20stroke%3D%22%239aa4b2%22%3E%3Cpath%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%20stroke-width%3D%221.5%22%20d%3D%22M6%208l4%204%204-4%22%2F%3E%3C%2Fsvg%3E')] bg-[position:right_0.75rem_center] bg-[length:1.25rem_1.25rem] bg-no-repeat pr-10`}
                value={draftFilters.interval}
                onChange={(event) => {
                  const nextInterval = event.target.value as DashboardInterval
                  setDraftFilters((current) =>
                    normalizeDashboardFilters({
                      ...current,
                      interval: nextInterval,
                      ...(nextInterval !== 'custom' ? { from: '', to: '' } : {}),
                    }),
                  )
                }}
              >
                {periods.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </div>

            {draftFilters.interval === 'custom' && (
              <>
                <div className="min-w-[140px] flex-1">
                  <label className={fieldLabelClass} htmlFor="from">
                    De
                  </label>
                  <input
                    id="from"
                    className={`${fieldClass} text-sm`}
                    type="date"
                    value={draftFilters.from}
                    onChange={(event) => setDraftFilters((current) => ({ ...current, from: event.target.value }))}
                  />
                </div>
                <div className="min-w-[140px] flex-1">
                  <label className={fieldLabelClass} htmlFor="to">
                    Até
                  </label>
                  <input
                    id="to"
                    className={`${fieldClass} text-sm`}
                    type="date"
                    value={draftFilters.to}
                    onChange={(event) => setDraftFilters((current) => ({ ...current, to: event.target.value }))}
                  />
                </div>
                <button
                  className={`${buttonMutedClass} px-3 text-sm shrink-0`}
                  type="button"
                  onClick={() => {
                    setDraftFilters((current) => ({ ...current, from: '', to: '' }))
                  }}
                  title="Limpar datas"
                >
                  <RefreshCcw className="size-4" />
                </button>
              </>
            )}

            <button className={`${buttonMutedClass} w-full sm:w-auto text-sm shrink-0`} type="button" onClick={() => void fetchSummary(draftFilters)} disabled={loading}>
              {loading ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
              Buscar
            </button>
          </div>
        </section>

        <section className={`${panelClass} rounded-[8px] p-5`}>
          <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <p className="text-xs font-bold uppercase tracking-widest text-muted">Painel de Métricas</p>
              <h2 className="mt-2 text-2xl font-bold text-text">Leads recebidos</h2>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-muted">
                Acompanhe o fluxo de recebimento de leads da UY3.
              </p>
            </div>

            <button
              className={`${buttonPrimaryClass} w-full text-sm font-medium sm:w-auto`}
              type="button"
              onClick={() => void handleExport()}
              disabled={exporting || !hasLeads || loading}
            >
              {exporting ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
              Baixar CSV
            </button>
          </div>

          <div className="grid gap-4 lg:grid-cols-[minmax(0,1.3fr)_minmax(18rem,0.7fr)]">
            <div className="rounded-[8px] border border-border bg-surface-soft p-6 flex flex-col justify-between">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <p className="text-xs font-bold uppercase tracking-widest text-muted">Total no período</p>
                  <p className="mt-4 text-5xl font-bold leading-none text-text">
                    {loading ? '--' : formatInteger(total)}
                  </p>
                  <p className="mt-3 text-sm text-muted">
                    {loading
                      ? 'Carregando métricas...'
                      : hasLeads
                        ? 'Leads processados no período.'
                        : 'Nenhum lead encontrado para os filtros.'}
                  </p>
                </div>

                <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full border border-border bg-[#10141b] text-accent">
                  <Database className="size-6" />
                </div>
              </div>

              <div className="mt-6 flex items-center gap-2 border-t border-[rgb(255_255_255_/_5%)] pt-4">
                <span className="text-xs font-bold uppercase tracking-widest text-muted">Último recebimento:</span>
                <span className="text-sm font-medium text-[#d7dde6]">
                  {loading ? '--' : data?.last_lead_at ? formatDateTime(data.last_lead_at) : 'Nenhum registro'}
                </span>
              </div>
            </div>

            <div className="rounded-[8px] border border-border bg-[#10141b] p-6 flex flex-col justify-center">
              <p className="text-xs font-bold uppercase tracking-widest text-muted">Filtro de Data Ativo</p>
              <p className="mt-3 text-lg font-bold text-text">{describeActiveRange(appliedFilters.interval, appliedFilters.from, appliedFilters.to)}</p>
            </div>
          </div>
        </section>
      </main>

      {logoutDialogOpen && (
        <div
          className="fixed inset-0 z-50 grid place-items-center bg-[rgb(5_7_10_/_62%)] p-6 backdrop-blur-[10px]"
          role="presentation"
          onClick={() => setLogoutDialogOpen(false)}
        >
          <div
            className="w-full max-w-[28rem] rounded-[10px] border border-[rgb(255_255_255_/_7%)] bg-[linear-gradient(180deg,rgb(21_25_34_/_98%),rgb(17_21_29_/_98%))] shadow-[inset_0_1px_0_rgb(255_255_255_/_5%),0_24px_64px_rgb(0_0_0_/_42%)]"
            role="dialog"
            aria-modal="true"
            aria-labelledby="logout-dialog-title"
            aria-describedby="logout-dialog-description"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="px-[1.4rem] pt-[1.4rem] pb-4">
              <h2 id="logout-dialog-title" className="m-0 text-[1.2rem] font-bold text-text">
                Confirmar Saída
              </h2>
              <p id="logout-dialog-description" className="mt-3 text-[0.96rem] leading-[1.5] text-muted">
                Tem certeza que deseja encerrar sua sessão?
              </p>
            </div>

            <div className="flex flex-col-reverse gap-3 border-t border-[rgb(255_255_255_/_5%)] px-[1.4rem] pt-4 pb-[1.35rem] sm:flex-row sm:justify-end sm:gap-3">
              <button
                className={`${buttonMutedClass} w-full sm:w-auto`}
                type="button"
                onClick={() => setLogoutDialogOpen(false)}
              >
                Cancelar
              </button>
              <button
                className={`${buttonDangerClass} w-full sm:w-auto`}
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

function errorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) {
    return error.message
  }
  return fallback
}

function formatInteger(value: number) {
  return new Intl.NumberFormat('pt-BR').format(value)
}

function normalizeDashboardFilters(value: Partial<DashboardFilters> | null | undefined): DashboardFilters {
  const interval =
    value?.interval && periodValues.has(value.interval)
      ? value.interval
      : defaultDashboardFilters.interval

  const filters: DashboardFilters = {
    interval,
    from: typeof value?.from === 'string' ? value.from : '',
    to: typeof value?.to === 'string' ? value.to : '',
  }

  if (filters.interval !== 'custom') {
    filters.from = ''
    filters.to = ''
  }

  return filters
}

function loadStoredDashboardFilters(): DashboardFilters {
  try {
    const raw = localStorage.getItem(dashboardFiltersStorageKey)
    if (!raw) {
      return { ...defaultDashboardFilters }
    }
    return normalizeDashboardFilters(JSON.parse(raw) as Partial<DashboardFilters>)
  } catch {
    return { ...defaultDashboardFilters }
  }
}

function saveDashboardFilters(filters: DashboardFilters) {
  try {
    localStorage.setItem(dashboardFiltersStorageKey, JSON.stringify(filters))
  } catch {
    // Mantem o dashboard funcional mesmo sem persistencia local.
  }
}

function formatDateTime(value: string) {
  if (!value) {
    return '-'
  }

  const normalized = value.trim()
  const match = normalized.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}:\d{2}:\d{2})$/)
  if (match) {
    return `${match[3]}/${match[2]}/${match[1]} ${match[4]}`
  }

  const date = new Date(normalized.replace(' ', 'T'))
  if (Number.isNaN(date.getTime())) {
    return normalized
  }

  return date.toLocaleString('pt-BR')
}

function describeActiveRange(interval: string, from: string, to: string) {
  if (interval === 'custom') {
    if (!from && !to) {
      return 'Todo o histórico'
    }
    if (from && to) {
      return `${from} até ${to}`
    }
    if (from) {
      return `A partir de ${from}`
    }
    if (to) {
      return `Até ${to}`
    }
    return 'Personalizado'
  }

  switch (interval) {
    case '24h':
      return 'Últimas 24 horas'
    case '7d':
      return 'Últimos 7 dias'
    case '30d':
      return 'Últimos 30 dias'
    case '90d':
      return 'Últimos 90 dias'
    default:
      return 'Todo o histórico'
  }
}

export default App
