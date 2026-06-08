import { useState } from 'react';
import { Play, Pause, TrendingUp, TrendingDown, DollarSign, BarChart3 } from 'lucide-react';
import { useFetch } from '../hooks/useApi';
import * as api from '../lib/api';
import { Toast } from '../components/Toast';
import type { BotStatus, Metrics, Portfolio } from '../types';

export default function Dashboard() {
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  const { data: statusData, refetch: refetchStatus } = useFetch(() => api.getBotStatus().then(r => r.data as BotStatus), 5000);
  const { data: portfolioData, refetch: refetchPortfolio } = useFetch(() => api.getPaperBalance().then(r => r.data as Portfolio), 10000);
  const { data: metricsData, refetch: refetchMetrics } = useFetch(() => api.getMetrics().then(r => r.data as Metrics), 10000);

  const showToast = (message: string, type: 'success' | 'error') => {
    setToast({ message, type });
  };

  const handleStart = async () => {
    try {
      await api.startBot();
      showToast('Bot gestart!', 'success');
      refetchStatus();
    } catch (e) {
      showToast('Kon bot niet starten: ' + (e instanceof Error ? e.message : ''), 'error');
    }
  };

  const handleStop = async () => {
    try {
      await api.stopBot();
      showToast('Bot gestopt!', 'success');
      refetchStatus();
    } catch (e) {
      showToast('Kon bot niet stoppen: ' + (e instanceof Error ? e.message : ''), 'error');
    }
  };

  const handleReset = async () => {
    try {
      await api.resetPaperBalance();
      showToast('Portfolio gereset!', 'success');
      refetchPortfolio();
      refetchMetrics();
    } catch (e) {
      showToast('Reset mislukt: ' + (e instanceof Error ? e.message : ''), 'error');
    }
  };

  const status = statusData;
  const portfolio = portfolioData;
  const metrics = metricsData;
  const pnl = portfolio?.profit_loss ?? 0;

  return (
    <div>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      <h1 className="text-2xl font-bold mb-6">Dashboard</h1>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatCard
          title="Balance"
          value={`€${(portfolio?.total_value ?? 0).toFixed(2)}`}
          icon={<DollarSign size={20} />}
        />
        <StatCard
          title="P&L"
          value={`${pnl >= 0 ? '+' : ''}€${Math.abs(pnl).toFixed(2)}`}
          icon={pnl >= 0 ? <TrendingUp size={20} /> : <TrendingDown size={20} />}
          color={pnl >= 0 ? 'text-green-400' : 'text-red-400'}
        />
        <StatCard
          title="Total Trades"
          value={String(metrics?.total_trades ?? 0)}
          icon={<BarChart3 size={20} />}
        />
        <StatCard
          title="Win Rate"
          value={`${((metrics?.win_rate ?? 0) * 100).toFixed(0)}%`}
          icon={<TrendingUp size={20} />}
          color="text-green-400"
        />
      </div>

      {/* Bot Controls */}
      <div className="card p-6 mb-8">
        <div className="flex items-center justify-between flex-wrap gap-4">
          <div>
            <h2 className="font-bold text-lg">Trading Bot</h2>
            <p className="text-gray-400 text-sm">
              Status: <span className={status?.is_running ? 'text-green-400' : 'text-gray-400'}>
                {status?.is_running ? 'Running' : 'Stopped'}
              </span>
              {status?.mode && ` • ${status.mode === 'paper' ? 'Paper Trading' : 'Live Trading'}`}
            </p>
          </div>
          <div className="flex gap-3">
            {!status?.is_running ? (
              <button onClick={handleStart} className="btn-primary flex items-center gap-2">
                <Play size={18} /> Start Bot
              </button>
            ) : (
              <button onClick={handleStop} className="btn-secondary flex items-center gap-2 !bg-red-500/20 !border-red-500/50 !text-red-400">
                <Pause size={18} /> Stop Bot
              </button>
            )}
            <button onClick={handleReset} className="btn-secondary">
              Reset Portfolio
            </button>
          </div>
        </div>
      </div>

      {/* Portfolio Details */}
      {portfolio && Object.keys(portfolio.token_balances).length > 0 && (
        <div className="card p-6">
          <h2 className="font-bold text-lg mb-4">Holdings</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="text-gray-400 text-sm border-b border-white/10">
                  <th className="pb-2">Token</th>
                  <th className="pb-2 text-right">Amount</th>
                  <th className="pb-2 text-right">Avg Price</th>
                  <th className="pb-2 text-right">Current Price</th>
                  <th className="pb-2 text-right">Value</th>
                </tr>
              </thead>
              <tbody>
                {Object.values(portfolio.token_balances).map((tb) => (
                  <tr key={tb.token.address} className="border-t border-white/5">
                    <td className="py-3">
                      <div className="font-semibold">{tb.token.symbol}</div>
                      <div className="text-sm text-gray-400">{tb.token.name}</div>
                    </td>
                    <td className="py-3 text-right">{tb.balance.toFixed(2)}</td>
                    <td className="py-3 text-right">€{tb.avg_price.toFixed(6)}</td>
                    <td className="py-3 text-right">€{tb.token.price.toFixed(6)}</td>
                    <td className="py-3 text-right">€{tb.value.toFixed(2)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

function StatCard({ title, value, icon, color = 'text-white' }: { title: string; value: string; icon: React.ReactNode; color?: string }) {
  return (
    <div className="card p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="text-gray-400 text-sm">{title}</span>
        <span className="text-gray-500">{icon}</span>
      </div>
      <div className={`text-2xl font-bold ${color}`}>{value}</div>
    </div>
  );
}
