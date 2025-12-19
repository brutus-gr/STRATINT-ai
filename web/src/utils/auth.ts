// Authentication utilities for making authenticated API requests

export function getAuthHeaders(): HeadersInit {
  const token = localStorage.getItem('admin_token');
  if (!token) {
    return {
      'Content-Type': 'application/json',
    };
  }

  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
}

export function getAuthToken(): string | null {
  return localStorage.getItem('admin_token');
}

export function setAuthToken(token: string): void {
  localStorage.setItem('admin_token', token);
}

export function clearAuthToken(): void {
  localStorage.removeItem('admin_token');
}

export function isAuthenticated(): boolean {
  return !!localStorage.getItem('admin_token');
}
