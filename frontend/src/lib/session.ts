const EMAIL_STORAGE_KEY = 'auth.email';

export function storeEmail(email: string) {
  window.localStorage.setItem(EMAIL_STORAGE_KEY, email);
}

export function getStoredEmail(): string | null {
  return window.localStorage.getItem(EMAIL_STORAGE_KEY);
}
