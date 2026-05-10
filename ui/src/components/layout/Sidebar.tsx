import { Link, useParams } from 'react-router-dom'
import { GitBranch, GitCommit, Code } from 'lucide-react'

interface SidebarProps {
  owner: string
  repo: string
  activeTab?: 'code' | 'commits' | 'branches'
}

export function RepoSidebar({ owner, repo, activeTab = 'code' }: SidebarProps) {
  const tabs = [
    { id: 'code', label: 'Code', icon: Code, href: `/${owner}/${repo}` },
    { id: 'commits', label: 'Commits', icon: GitCommit, href: `/${owner}/${repo}/commits/main` },
    { id: 'branches', label: 'Branches', icon: GitBranch, href: `/${owner}/${repo}/branches` },
  ] as const

  return (
    <div className="flex border-b border-line">
      {tabs.map((tab) => {
        const Icon = tab.icon
        const active = activeTab === tab.id
        return (
          <Link
            key={tab.id}
            to={tab.href}
            className={`flex items-center gap-1.5 px-4 py-2.5 text-sm border-b-2 transition-colors no-underline ${
              active
                ? 'border-accent-orange text-primary'
                : 'border-transparent text-secondary hover:text-primary hover:border-line'
            }`}
          >
            <Icon size={14} />
            {tab.label}
          </Link>
        )
      })}
    </div>
  )
}

// Re-export useParams for convenience
export { useParams }
