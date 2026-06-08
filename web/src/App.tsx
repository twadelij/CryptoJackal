import { Routes, Route, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { LayoutDashboard, Search, Wallet, History, Settings, Activity, HelpCircle, LogOut } from 'lucide-react'
import { useEffect } from 'react'
import Dashboard from './pages/Dashboard'
import SetupWizard from './pages/SetupWizard'
import Tokens from './pages/Tokens'
import Portfolio from './pages/Portfolio'
import TradeHistory from './pages/TradeHistory'
import Help from './pages/Help'
import Login from './pages/Login'
import { logout } from './lib/api'
import * as api from './lib/api'

function App() {
  const navigate = useNavigate()
  const location = useLocation()
  const token = localStorage.getItem('cj_token')
  const isLoginPage = location.pathname === '/login'

  useEffect(() => {
    // Redirect to login if no token and not already on login page
    if (!token && !isLoginPage) {
      navigate('/login')
      return
    }

    // Redirect to setup if live mode is active but no ETH node configured
    if (token && location.pathname !== '/setup' && location.pathname !== '/login') {
      api.getConfig().then(res => {
        if (res.success && res.data && !res.data.paper_trading_mode && !res.data.eth_node_url) {
          navigate('/setup')
        }
      }).catch(() => {})
    }
  }, [location.pathname, navigate, token, isLoginPage])

  // Show login page without sidebar
  if (isLoginPage) {
    return (
      <Routes>
        <Route path="/login" element={<Login />} />
      </Routes>
    )
  }

  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <nav className="w-16 md:w-56 bg-jackal-dark/80 border-r border-white/10 flex flex-col">
        <div className="p-4 flex items-center gap-3">
          <img src="/logo.png" alt="CryptoJackal" className="w-10 h-10 rounded-lg" />
          <span className="font-bold text-lg hidden md:block">CryptoJackal</span>
        </div>

        <div className="flex-1 py-4 space-y-1">
          <NavItem to="/" icon={<LayoutDashboard size={20} />} label="Dashboard" />
          <NavItem to="/tokens" icon={<Search size={20} />} label="Tokens" />
          <NavItem to="/portfolio" icon={<Wallet size={20} />} label="Portfolio" />
          <NavItem to="/history" icon={<History size={20} />} label="History" />
          <NavItem to="/setup" icon={<Settings size={20} />} label="Setup" />
          <NavItem to="/help" icon={<HelpCircle size={20} />} label="Help" />
        </div>

        <div className="p-4 border-t border-white/10 space-y-2">
          <button
            onClick={logout}
            className="flex items-center gap-3 px-4 py-2 mx-2 w-[calc(100%-1rem)] rounded-lg text-gray-400 hover:text-white hover:bg-white/5 transition-colors"
          >
            <LogOut size={20} />
            <span className="hidden md:block">Logout</span>
          </button>
          <div className="flex items-center gap-2 text-sm text-gray-400 px-4">
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
          <Route path="/help" element={<Help />} />
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

export default App
