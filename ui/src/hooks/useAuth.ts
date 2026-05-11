import { useAuthStore } from '../store/auth'

/**
 * Convenience wrapper — returns the full auth store.
 * Rehydration is handled exclusively by useAuthRehydration() in App.tsx.
 * Pages and components never need their own rehydration checks.
 */
export function useAuth() {
  return useAuthStore()
}
