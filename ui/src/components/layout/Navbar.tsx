import { useState, useRef, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Anvil, Plus, Bell, ChevronDown, Search } from 'lucide-react'
import { useAuthStore } from '../../store/auth'
import { Avatar } from '../ui/Avatar'

export function Navbar() {
  const { user, isAuthenticated, logout } = useAuthStore()
  const navigate = useNavigate()
  const [userMenuOpen, setUserMenuOpen] = useState(false)
  const [plusMenuOpen, setPlusMenuOpen] = useState(false)
  const userMenuRef = useRef<HTMLDivElement>(null)
  const plusMenuRef = useRef<HTMLDivElement>(null)

  // Close dropdowns on outside click
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false)
      }
      if (plusMenuRef.current && !plusMenuRef.current.contains(e.target as Node)) {
        setPlusMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  function handleLogout() {
    logout()
    navigate('/login')
  }

  return (
    <nav
      className="fixed top-0 left-0 right-0 z-50 flex items-center gap-4 px-4 h-12"
      style={{
        background: 'rgba(10, 11, 13, 0.95)',
        backdropFilter: 'blur(8px)',
        borderBottom: '1px solid #2a2d35',
      }}
    >
      {/* Logo */}
      <Link
        to="/"
        className="flex items-center gap-2 flex-shrink-0 text-accent-blue hover:text-[#79c0ff] transition-colors no-underline"
      >
        <Anvil size={18} strokeWidth={1.5} />
        <span className="font-mono font-semibold text-sm tracking-tight">GForce</span>
      </Link>

      {/* Search */}
      <div className="flex-1 max-w-xs relative">
        <Search
          size={13}
          className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted pointer-events-none"
        />
        <input
          type="text"
          placeholder="Search repositories..."
          className="w-full h-7 bg-base border border-line text-primary placeholder:text-muted font-mono text-xs pl-8 pr-3 outline-none focus:border-accent-blue focus:ring-1 focus:ring-accent-blue transition-colors"
        />
      </div>

      <div className="flex-1" />

      {isAuthenticated ? (
        <div className="flex items-center gap-1">
          {/* New (+) menu */}
          <div ref={plusMenuRef} className="relative">
            <button
              onClick={() => setPlusMenuOpen(!plusMenuOpen)}
              className="flex items-center gap-0.5 h-7 px-2 text-secondary hover:text-primary transition-colors"
            >
              <Plus size={16} />
              <ChevronDown size={11} />
            </button>
            {plusMenuOpen && (
              <div
                className="absolute right-0 top-full mt-1 w-44 bg-overlay border border-line py-1 z-50 animate-slide-down"
                style={{ boxShadow: '0 8px 24px rgba(0,0,0,0.5)' }}
              >
                <Link
                  to="/new"
                  className="flex items-center px-3 py-1.5 text-sm text-primary hover:bg-elevated no-underline transition-colors"
                  onClick={() => setPlusMenuOpen(false)}
                >
                  New repository
                </Link>
              </div>
            )}
          </div>

          {/* Bell */}
          <button className="h-7 w-7 flex items-center justify-center text-secondary hover:text-primary transition-colors">
            <Bell size={16} />
          </button>

          {/* User avatar + menu */}
          <div ref={userMenuRef} className="relative ml-1">
            <button
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex items-center"
            >
              <Avatar user={user} size="xs" />
            </button>
            {userMenuOpen && (
              <div
                className="absolute right-0 top-full mt-1 w-48 bg-overlay border border-line py-1 z-50 animate-slide-down"
                style={{ boxShadow: '0 8px 24px rgba(0,0,0,0.5)' }}
              >
                <div className="px-3 py-2 border-b border-line-muted">
                  <p className="text-xs text-secondary">Signed in as</p>
                  <p className="text-sm font-mono text-primary font-medium">{user?.username}</p>
                </div>
                <Link
                  to={`/${user?.username}`}
                  className="flex items-center px-3 py-1.5 text-sm text-primary hover:bg-elevated no-underline transition-colors"
                  onClick={() => setUserMenuOpen(false)}
                >
                  Your profile
                </Link>
                <Link
                  to="/repos"
                  className="flex items-center px-3 py-1.5 text-sm text-primary hover:bg-elevated no-underline transition-colors"
                  onClick={() => setUserMenuOpen(false)}
                >
                  Your repositories
                </Link>
                <div className="border-t border-line-muted mt-1 pt-1">
                  <button
                    onClick={handleLogout}
                    className="w-full text-left px-3 py-1.5 text-sm text-accent-red hover:bg-elevated transition-colors"
                  >
                    Sign out
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="flex items-center gap-2">
          <Link
            to="/login"
            className="h-7 px-3 text-sm text-secondary hover:text-primary border border-line hover:border-[#3a3d45] flex items-center transition-colors no-underline"
          >
            Sign in
          </Link>
          <Link
            to="/register"
            className="h-7 px-3 text-sm bg-accent-blue text-base hover:bg-[#79c0ff] flex items-center transition-colors no-underline font-medium"
          >
            Register
          </Link>
        </div>
      )}
    </nav>
  )
}
