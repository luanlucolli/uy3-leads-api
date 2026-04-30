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

const pillBaseClass =
  'inline-flex items-center rounded-full bg-[#242b38] px-[0.55rem] py-[0.2rem] text-[0.76rem] font-bold whitespace-nowrap'

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
      <header className="sticky top-0 z-20 border-b border-border bg-[rgb(15_17_21_/_84%)] backdrop-blur-[16px]">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-3">
            <div className={brandMarkClass}>
              <Database className="size-6" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-text">Painel UY3</h1>
              <p className="hidden text-sm text-muted sm:block">Leads recebidos via webhook</p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <span className="hidden text-sm font-medium text-muted md:inline">{user.email}</span>
            <button className={`${buttonMutedClass} text-sm font-medium`} type="button" onClick={() => setLogoutDialogOpen(true)}>
              <LogOut className="size-4" />
              Sair
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:py-8">
        <section className={`${panelClass} mb-6 rounded-[8px]`}>
          <div className="flex items-center justify-between border-b border-border bg-surface-soft/30 px-5 py-4">
            <h2 className="flex items-center gap-2 text-base font-semibold text-text">
              <Filter className="size-5 text-muted" />
              Filtros e Exportação
            </h2>
            <div className="flex items-center gap-2 text-sm font-medium text-muted">
              <span>Total filtrado:</span>
              <span className="rounded-md bg-border/50 px-2 py-0.5 text-text">
                {formatInteger(data?.total ?? 0)}
              </span>
            </div>
          </div>
          
          <div className="p-5">
            <div className="flex flex-col gap-5 sm:flex-row sm:flex-wrap sm:items-end">
              <div className="min-w-[200px] flex-1">
                <label className={`${fieldLabelClass} mb-1 text-sm font-medium`} htmlFor="interval">
                  Intervalo
                </label>
                <select
                  id="interval"
                  className={`${fieldClass} text-sm`}
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
                <label className={`${fieldLabelClass} mb-1 text-sm font-medium`} htmlFor="order">
                  Ordenação
                </label>
                <select id="order" className={`${fieldClass} text-sm`} value={direction} onChange={(event) => setDirection(event.target.value as 'asc' | 'desc')}>
                  <option value="desc">Mais recente</option>
                  <option value="asc">Mais antigo</option>
                </select>
              </div>

              {interval === 'custom' && (
                <div className="flex w-full flex-col gap-4 sm:w-auto sm:flex-1 sm:flex-row sm:items-end md:min-w-[340px]">
                  <div className="flex-1">
                    <label className={`${fieldLabelClass} mb-1 text-sm font-medium`} htmlFor="from">
                      De
                    </label>
                    <input id="from" className={`${fieldClass} text-sm`} type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
                  </div>

                  <div className="flex-1">
                    <label className={`${fieldLabelClass} mb-1 text-sm font-medium`} htmlFor="to">
                      Até
                    </label>
                    <input id="to" className={`${fieldClass} text-sm`} type="date" value={to} onChange={(event) => setTo(event.target.value)} />
                  </div>

                  <button
                    className={`${buttonMutedClass} mt-2 text-sm sm:mt-0`}
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
              <button className={`${buttonMutedClass} w-full text-sm font-medium sm:w-auto`} type="button" onClick={() => void handleExport()} disabled={exporting}>
                {exporting ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                Exportar CSV
              </button>
              
              <button className={`${buttonPrimaryClass} w-full text-sm font-medium sm:w-auto`} type="button" onClick={() => void fetchLeads(1)} disabled={loading}>
                {loading ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
                Buscar Leads
              </button>
            </div>
          </div>
        </section>

        <section className={`${panelClass} overflow-hidden rounded-[8px]`}>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[980px] border-collapse text-[0.9rem]">
              <thead className="text-left font-semibold text-muted">
                <tr>
                  <th className="whitespace-nowrap bg-surface-soft px-4 py-[0.85rem] text-left text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted">ID</th>
                  <th className="whitespace-nowrap bg-surface-soft px-4 py-[0.85rem] text-left text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted">Recebido em</th>
                  <th className="whitespace-nowrap bg-surface-soft px-4 py-[0.85rem] text-left text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted">CPF</th>
                  <th className="whitespace-nowrap bg-surface-soft px-4 py-[0.85rem] text-left text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted">Nome</th>
                  <th className="whitespace-nowrap bg-surface-soft px-4 py-[0.85rem] text-left text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted">Status</th>
                  <th className="whitespace-nowrap bg-surface-soft px-4 py-[0.85rem] text-left text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted">Elegível</th>
                  <th className="whitespace-nowrap bg-surface-soft px-4 py-[0.85rem] text-right text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted">Valor liberado</th>
                  <th className="whitespace-nowrap bg-surface-soft px-4 py-[0.85rem] text-right text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted">Margem</th>
                  <th className="whitespace-nowrap bg-surface-soft px-4 py-[0.85rem] text-right text-[0.72rem] font-bold uppercase tracking-[0.04em] text-muted">Parcelas</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {loading && (
                  <tr>
                    <td className="h-40 py-8 text-center text-muted" colSpan={9}>
                      <Loader2 className="mr-2 inline size-5 animate-spin text-accent" />
                      Carregando leads
                    </td>
                  </tr>
                )}

                {!loading && leads.length === 0 && (
                  <tr>
                    <td className="h-40 py-8 text-center text-muted" colSpan={9}>
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

          <div className="flex flex-col gap-4 border-t border-border bg-surface-soft px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3 text-sm font-medium text-muted">
              <span>
                Página {data?.current_page ?? 1} de {Math.max(data?.total_pages ?? 1, 1)}
              </span>
              <select
                className={`${fieldClass} h-9 w-20 text-sm`}
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
              <button className={`${buttonMutedClass} text-sm font-medium`} type="button" disabled={!data?.has_previous || loading} onClick={() => void fetchLeads(page - 1)}>
                <ChevronLeft className="size-4" />
                Anterior
              </button>
              <button className={`${buttonMutedClass} text-sm font-medium`} type="button" disabled={!data?.has_next || loading} onClick={() => void fetchLeads(page + 1)}>
                Próxima
                <ChevronRight className="size-4" />
              </button>
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

function LeadRow({ lead }: { lead: Lead }) {
  return (
    <tr className="hover:bg-[rgb(255_255_255_/_2.5%)]">
      <td className="border-t border-border py-[0.85rem] px-4 align-middle font-mono text-xs text-muted">#{lead.id}</td>
      <td className="border-t border-border whitespace-nowrap px-4 py-[0.85rem] align-middle">{formatDateTime(lead.received_at)}</td>
      <td className="border-t border-border px-4 py-[0.85rem] align-middle font-mono text-sm">{formatCPF(lead.cpf)}</td>
      <td className="min-w-52 border-t border-border px-4 py-[0.85rem] align-middle font-medium">{lead.nome_trabalhador || '-'}</td>
      <td className="border-t border-border px-4 py-[0.85rem] align-middle">
        <span className={`${pillBaseClass} text-xs font-medium`}>{lead.status || '-'}</span>
      </td>
      <td className="border-t border-border px-4 py-[0.85rem] align-middle">
        <span className={boolLabel(lead.elegivel_emprestimo) === 'Sim' ? `${pillBaseClass} bg-[rgb(45_212_191_/_14%)] text-[#c8fff7] text-xs font-medium` : `${pillBaseClass} text-xs font-medium text-muted`}>{boolLabel(lead.elegivel_emprestimo)}</span>
      </td>
      <td className="border-t border-border whitespace-nowrap px-4 py-[0.85rem] text-right align-middle font-medium">{formatBRL(lead.valor_liberado)}</td>
      <td className="border-t border-border whitespace-nowrap px-4 py-[0.85rem] text-right align-middle">{formatBRL(lead.margem_disponivel)}</td>
      <td className="border-t border-border px-4 py-[0.85rem] text-right align-middle">{lead.numero_parcelas || '-'}</td>
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
