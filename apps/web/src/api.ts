// API client for the QuoteWise Go API.
//
// Contracts (ADR-007): the session cookie is HttpOnly (never read
// here — the browser carries it), and every mutating request sends
// the X-Requested-With header required by the CSRF middleware. The
// browser only ever talks to the Go API — never the AI service.

export interface User {
  id: string;
  email: string;
  created_at: string;
}

export interface Organization {
  id: string;
  name: string;
  created_at: string;
}

export interface Me {
  user: User;
  organization: Organization;
  role: string;
}

export interface Member {
  user_id: string;
  email: string;
  role: string;
  joined_at: string;
}

export interface AuthResponse {
  user: User;
  organization?: Organization;
}

export interface Health {
  status: string;
  version: string;
  checks?: Record<string, { status: string; detail?: string }>;
}

/** Typed API error carrying the server's error-envelope code. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

interface ErrorEnvelope {
  error?: { code?: string; message?: string };
}

const CSRF_HEADER = "X-Requested-With";
const CSRF_VALUE = "XMLHttpRequest";

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const method = options.method ?? "GET";
  const headers = new Headers(options.headers);
  if (options.body) {
    headers.set("Content-Type", "application/json");
  }
  if (method !== "GET" && method !== "HEAD") {
    headers.set(CSRF_HEADER, CSRF_VALUE);
  }

  const response = await fetch(path, { ...options, method, headers, credentials: "same-origin" });
  if (!response.ok) {
    let code = "error";
    let message = `Request failed with HTTP ${response.status}`;
    try {
      const body = (await response.json()) as ErrorEnvelope;
      if (body.error) {
        code = body.error.code ?? code;
        message = body.error.message ?? message;
      }
    } catch {
      // Non-JSON error body — keep the generic message.
    }
    throw new ApiError(response.status, code, message);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

export const api = {
  /** Register: creates the account, its organization, and a session. */
  register(input: { email: string; password: string; organization_name: string }): Promise<AuthResponse> {
    return request<AuthResponse>("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  /** Sign in with email + password. */
  login(email: string, password: string): Promise<AuthResponse> {
    return request<AuthResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  },

  /** Sign out (revokes the session server-side; clears the cookie). */
  logout(): Promise<void> {
    return request<void>("/api/v1/auth/logout", { method: "POST" });
  },

  /** Current user + organization + role; 401 when signed out. */
  me(): Promise<Me> {
    return request<Me>("/api/v1/me");
  },

  /** Members of the current organization (session-scoped server-side). */
  members(): Promise<{ members: Member[] }> {
    return request<{ members: Member[] }>("/api/v1/organizations/current/members");
  },

  /** Go API liveness. */
  health(): Promise<Health> {
    return request<Health>("/healthz");
  },

  /** Go API readiness (dependency checks). */
  ready(): Promise<Health> {
    return request<Health>("/readyz");
  },
};
