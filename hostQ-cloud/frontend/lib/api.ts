// Thin fetch client. Holds the in-memory access token, auto-refreshes when
// the API returns 401, exposes a typed JSON helper.

let accessToken: string | null = null;

export function setAccessToken(t: string | null) {
  accessToken = t;
}

export function getAccessToken() {
  return accessToken;
}

type FetchOpts = Omit<RequestInit, "body"> & {
  json?: unknown;
};

class ApiError extends Error {
  status: number;
  data: unknown;
  constructor(status: number, message: string, data: unknown) {
    super(message);
    this.status = status;
    this.data = data;
  }
}

async function rawFetch(path: string, opts: FetchOpts = {}): Promise<Response> {
  const headers = new Headers(opts.headers || {});
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (opts.json !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(path, {
    ...opts,
    credentials: "include",
    headers,
    body: opts.json !== undefined ? JSON.stringify(opts.json) : (opts as any).body,
  });
}

// On 401 the client tries one refresh, then retries the original call.
async function withRefresh(path: string, opts: FetchOpts): Promise<Response> {
  let res = await rawFetch(path, opts);
  if (res.status !== 401) return res;
  const refresh = await rawFetch("/api/auth/refresh", { method: "POST" });
  if (!refresh.ok) return res;
  const { accessToken: newToken } = (await refresh.json()) as { accessToken: string };
  setAccessToken(newToken);
  res = await rawFetch(path, opts);
  return res;
}

export async function apiJSON<T = unknown>(path: string, opts: FetchOpts = {}): Promise<T> {
  const res = await withRefresh(path, opts);
  const text = await res.text();
  const data = text ? (() => { try { return JSON.parse(text); } catch { return text; } })() : null;
  if (!res.ok) {
    const msg = (data && typeof data === "object" && "error" in (data as any)) ? (data as any).error : res.statusText;
    throw new ApiError(res.status, msg, data);
  }
  return data as T;
}

export { ApiError };
