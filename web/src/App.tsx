import { Routes, Route, NavLink } from 'react-router-dom'
import { LayoutDashboard, Search, Wallet, History, Settings, Activity } from 'lucide-react'
import Dashboard from './pages/Dashboard'
import SetupWizard from './pages/SetupWizard'
import Tokens from './pages/Tokens'
import Portfolio from './pages/Portfolio'
import TradeHistory from './pages/TradeHistory'

function Layout() {
  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <nav className="w-16 md:w-56 bg-jackal-dark/80 border-r border-white/10 flex flex-col">
        <div className="p-4 flex items-center gap-3">
          <div className="w-10 h-10 bg-gradient-to-br from-yellow-400 to-orange-500 rounded-lg flex items-center justify-center">
            <span className="text-xl">🐺</span>
          </div>
          <span className="font-bold text-lg hidden md:block">CryptoJackal</span>
        </div>

        <div className="flex-1 py-4 space-y-1">
          <NavItem to="/" icon={<LayoutDashboard size={20} />} label="Dashboard" />
          <NavItem to="/tokens" icon={<Search size={20} />} label="Tokens" />
          <NavItem to="/portfolio" icon={<Wallet size={20} />} label="Portfolio" />
          <NavItem to="/history" icon={<History size={20} />} label="History" />
          <NavItem to="/setup" icon={<Settings size={20} />} label="Setup" />
        </div>

        <div className="p-4 border-t border-white/10">
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <Activity size={16} />
            <span className="hidden md:inline">v1.0.0</span>
          </div>
        </div>
      </nav>

      {/* Main content */}
      <main className="flex-1 p-6 overflow-auto">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/setup" element={<SetupWizard />} />
          <Route path="/tokens" element={<Tokens />} />
          <Route path="/portfolio" element={<Portfolio />} />
          <Route path="/history" element={<TradeHistory />} />
        </Routes>
      </main>
    </div>
  )
}

function NavItem({ to, icon, label }: { to: string; icon: React.ReactNode; label: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `flex items-center gap-3 px-4 py-3 mx-2 rounded-lg transition-colors ${
          isActive
            ? 'bg-jackal-pink/20 text-jackal-pink'
            : 'text-gray-400 hover:text-white hover:bg-white/5'
        }`
      }
    >
      {icon}
      <span className="hidden md:block">{label}</span>
    </NavLink>
  )
}

export default Layout
