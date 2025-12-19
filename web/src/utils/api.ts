// API configuration
// In production, use the same origin as the frontend
// In development, fall back to localhost:8080
export const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ||
  (import.meta.env.MODE === 'production'
    ? window.location.origin
    : 'http://localhost:8080');
