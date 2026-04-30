export interface User {
  id: number
  email: string
}

export interface Lead {
  id: number
  cpf: string
  nome_trabalhador: string
  status: string
  elegivel_emprestimo: string
  valor_liberado: number
  margem_disponivel: number
  numero_parcelas: number
  data_hora_validade_solicitacao: string
  data_nascimento: string
  data_admissao: string
  is_mei: string
  is_judicial_recovery: string
  pep_codigo: string
  active_fgts_debts: string
  type_webhook: string
  exportado: number
  received_at: string
}

export interface LeadsResponse {
  items: Lead[]
  total: number
  current_page: number
  per_page: number
  total_pages: number
  has_next: boolean
  has_previous: boolean
}

export interface LeadFilters {
  page?: number
  per_page?: number
  period?: string
  from?: string
  to?: string
  sort?: 'received_at' | 'id'
  direction?: 'asc' | 'desc'
}

const TOKEN_KEY = 'uy3_token'
const apiBase = (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const headers = new Headers(options.headers)

  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(`${apiBase}${path}`, {
    ...options,
    headers,
  })

  if (!response.ok) {
    let message = `Erro ${response.status}`
    try {
      const payload = await response.json()
      if (typeof payload.error === 'string') {
        message = payload.error
      }
    } catch {
      // Respostas de erro do export podem nao vir em JSON.
    }
    throw new ApiError(response.status, message)
  }

  return response.json() as Promise<T>
}

function buildQuery(params: LeadFilters) {
  const query = new URLSearchParams()

  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') {
      query.set(key, String(value))
    }
  })

  const qs = query.toString()
  return qs ? `?${qs}` : ''
}

export const api = {
  login(email: string, password: string) {
    return request<{ token: string }>('/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
  },

  me() {
    return request<User>('/me')
  },

  leads(filters: LeadFilters) {
    return request<LeadsResponse>(`/leads${buildQuery(filters)}`)
  },
}

export async function downloadLeadsExport(filters: LeadFilters) {
  const token = getToken()
  const response = await fetch(`${apiBase}/leads/export${buildQuery(filters)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  })

  if (!response.ok) {
    throw new ApiError(response.status, `Erro ${response.status}`)
  }

  const blob = await response.blob()
  const filename = exportFilename(response.headers.get('Content-Disposition'))
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function exportFilename(contentDisposition: string | null) {
  const match = contentDisposition?.match(/filename="([^"]+)"/)
  return match?.[1] ?? `leads_${new Date().toISOString().slice(0, 10)}.csv`
}
