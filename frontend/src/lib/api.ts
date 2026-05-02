export interface User {
  id: number
  email: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface LeadsSummaryResponse {
  total: number
  last_lead_at?: string
}

export interface LeadFilters {
  period?: string
  from?: string
  to?: string
}

const TOKEN_KEY = 'uy3_token'
const apiBase = (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')
const requestTimeoutMs = 35_000
const transientRetryDelaysMs = [1_500]
const transientStatuses = new Set([502, 503, 504])

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
  transient: boolean

  constructor(status: number, message: string, transient = false) {
    super(message)
    this.status = status
    this.transient = transient
  }
}

export function isUnauthorizedApiError(error: unknown) {
  return error instanceof ApiError && error.status === 401
}

export function isTransientApiError(error: unknown) {
  return error instanceof ApiError && error.transient
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await performRequest(path, options)
  const text = await response.text()
  if (!text) {
    return {} as T
  }

  try {
    return JSON.parse(text) as T
  } catch {
    return text as T
  }
}

async function performRequest(path: string, options: RequestInit = {}) {
  const requestOptions = withAuthHeaders(options)

  for (let attempt = 0; attempt <= transientRetryDelaysMs.length; attempt += 1) {
    try {
      const response = await fetchWithTimeout(`${apiBase}${path}`, requestOptions)
      if (!response.ok) {
        const apiError = await buildApiError(response)
        if (shouldRetry(apiError, attempt)) {
          await delay(transientRetryDelaysMs[attempt])
          continue
        }
        throw apiError
      }

      return response
    } catch (error) {
      const apiError = normalizeRequestError(error)
      if (shouldRetry(apiError, attempt)) {
        await delay(transientRetryDelaysMs[attempt])
        continue
      }
      throw apiError
    }
  }

  throw new ApiError(0, transientServiceMessage(), true)
}

function withAuthHeaders(options: RequestInit) {
  const token = getToken()
  const headers = new Headers(options.headers)

  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  return {
    ...options,
    headers,
  }
}

async function fetchWithTimeout(input: string, options: RequestInit) {
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), requestTimeoutMs)
  let detachAbort: (() => void) | undefined

  if (options.signal) {
    if (options.signal.aborted) {
      controller.abort()
    } else {
      const forwardAbort = () => controller.abort()
      options.signal.addEventListener('abort', forwardAbort, { once: true })
      detachAbort = () => options.signal?.removeEventListener('abort', forwardAbort)
    }
  }

  try {
    return await fetch(input, {
      ...options,
      signal: controller.signal,
    })
  } finally {
    window.clearTimeout(timeoutId)
    detachAbort?.()
  }
}

async function buildApiError(response: Response) {
  const contentType = response.headers.get('Content-Type') ?? ''
  let message = friendlyMessageForStatus(response.status)

  if (contentType.includes('application/json')) {
    try {
      const payload = await response.json()
      if (payload && typeof payload.error === 'string' && payload.error.trim() !== '') {
        message = payload.error.trim()
      }
    } catch {
      // Mantemos a mensagem amigavel padrao quando o JSON vem invalido.
    }
  }

  return new ApiError(response.status, message, transientStatuses.has(response.status))
}

function normalizeRequestError(error: unknown) {
  if (error instanceof ApiError) {
    return error
  }

  if (error instanceof DOMException && error.name === 'AbortError') {
    return new ApiError(0, transientServiceMessage(), true)
  }

  if (error instanceof TypeError) {
    return new ApiError(0, transientServiceMessage(), true)
  }

  if (error instanceof Error) {
    return new ApiError(0, error.message || transientServiceMessage(), true)
  }

  return new ApiError(0, transientServiceMessage(), true)
}

function shouldRetry(error: ApiError, attempt: number) {
  return error.transient && attempt < transientRetryDelaysMs.length
}

function friendlyMessageForStatus(status: number) {
  if (transientStatuses.has(status)) {
    return transientServiceMessage()
  }
  if (status >= 500) {
    return 'Serviço indisponível no momento. Tente novamente em instantes.'
  }

  return `Erro ${status}`
}

function transientServiceMessage() {
  return 'Serviço indisponível no momento. Ele pode estar acordando no plano gratuito.'
}

function delay(ms: number | undefined) {
  if (!ms) {
    return Promise.resolve()
  }

  return new Promise<void>((resolve) => {
    window.setTimeout(resolve, ms)
  })
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
    return request<LoginResponse>('/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
  },

  me() {
    return request<User>('/me')
  },

  leads(filters: LeadFilters) {
    return request<LeadsSummaryResponse>(`/leads${buildQuery(filters)}`)
  },
}

export async function downloadLeadsExport(filters: LeadFilters) {
  const response = await performRequest(`/leads/export${buildQuery(filters)}`)
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
