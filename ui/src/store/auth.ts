import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import type { User } from '../types/api'

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  /** True after Zustand has finished reading from localStorage. */
  isRehydrated: boolean
  login: (token: string, user: User) => void
  setUser: (user: User) => void
  /** Clears auth state. TanStack Query cache is cleared separately via App.tsx. */
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      isRehydrated: false,
      login: (token, user) =>
        set({ token, user, isAuthenticated: true }),
      setUser: (user) =>
        set({ user }),
      logout: () =>
        set({ token: null, user: null, isAuthenticated: false }),
    }),
    {
      name: 'gforce-auth',
      storage: createJSONStorage(() => localStorage),
      // Only persist the token — user object is re-fetched on app load.
      partialize: (state) => ({ token: state.token }),
      onRehydrateStorage: () => (_state, error) => {
        if (!error) {
          useAuthStore.setState({ isRehydrated: true })
        }
      },
    }
  )
)

/** True once Zustand has finished reading the persisted token from localStorage. */
export function useIsRehydrated(): boolean {
  return useAuthStore((s) => s.isRehydrated)
}
