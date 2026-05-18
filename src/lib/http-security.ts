export function allowInsecureHttp(): boolean {
  return process.env.HOSTQ_ALLOW_INSECURE_HTTP === 'true';
}

export function requestIsHttps(request: Request): boolean {
  const forwardedProto = request.headers.get('x-forwarded-proto');
  if (forwardedProto) return forwardedProto.split(',')[0]?.trim() === 'https';
  try {
    return new URL(request.url).protocol === 'https:';
  } catch {
    return false;
  }
}

export function shouldUseSecureCookies(request: Request): boolean {
  if (process.env.NODE_ENV !== 'production') return false;
  return requestIsHttps(request) || !allowInsecureHttp();
}
