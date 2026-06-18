/** Reads the current access token from the persisted auth store. */
export function getAccessToken(): string | null {
  try {
    const raw = localStorage.getItem('auth-storage');
    if (!raw) return null;
    return JSON.parse(raw)?.state?.accessToken ?? null;
  } catch {
    return null;
  }
}
