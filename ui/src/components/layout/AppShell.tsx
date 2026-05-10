import { Outlet } from 'react-router-dom'
import { Navbar } from './Navbar'

export function AppShell() {
  return (
    <div className="min-h-screen bg-base text-primary">
      <Navbar />
      {/* 48px offset for fixed navbar */}
      <main className="pt-12">
        <Outlet />
      </main>
    </div>
  )
}
