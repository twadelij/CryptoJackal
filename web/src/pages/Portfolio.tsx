import { useState } from 'react';
import { Wallet, TrendingUp, TrendingDown, ArrowUpRight } from 'lucide-react';
import { useFetch } from '../hooks/useApi';
import * as api from '../lib/api';
import { Toast } from '../components/Toast';
import type { Portfolio as PortfolioType } from '../types';

export default function Portfolio() {
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);
  const [sellToken, setSellToken] = useState<{ address: string; symbol: string; name: string; price: number; balance: number } | null>(null);
  const [sellAmount, setSellAmount] = useState(0);

  const { data: portfolioData, loading, refetch } = useFetch(
    () => api.getPaperBalance().then(r => r.data as PortfolioType),
    10000
  );

  const showToast = (message: string, type: 'success' | 'error') => {
    setToast({ message, type });
  };

  const handleSell = async () => {
    if (!sellToken) return;
    try {
      await api.executePaperTrade(
        sellToken.address,
        sellToken.symbol,
        sellToken.name,
        sellToken.price,
        sellAmount,
        'sell'
      );
      showToast(`Verkocht ${sellAmount} ${sellToken.symbol}`, 'success');
      setSellToken(null);
      refetch();
    } catch (e) {
      showToast('Verkoop mislukt: ' + (e instanceof Error ? e.message : ''), 'error');
    }
  };

  const portfolio = portfolioData;
  const pnl = portfolio?.profit_loss ?? 0;
  const holdings = portfolio ? Object.values(portfolio.token_balances) : [];

  return (
    <div>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      <h1 className="text-2xl font-bold mb-6">Portfolio</h1>

      {/* Overview Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
        <div className="card p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Totale Waarde</span>
            <Wallet size={18} className="text-gray-500" />
          </div>
          <div className="text-2xl font-bold">€{(portfolio?.total_value ?? 0).toFixed(2)}</div>
        </div>
        <div className="card p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">P&L</span>
            {pnl >= 0 ? <TrendingUp size={18} className="text-green-400" /> : <TrendingDown size={18} className="text-red-400" />}
          </div>
          <div className={`text-2xl font-bold ${pnl >= 0 ? 'text-green-400' : 'text-red-400'}`}>
            {pnl >= 0 ? '+' : ''}€{Math.abs(pnl).toFixed(2)}
          </div>
          <div className="text-sm text-gray-400">{portfolio?.profit_loss_pct?.toFixed(2)}%</div>
        </div>
        <div className="card p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Cash</span>
            <ArrowUpRight size={18} className="text-gray-500" />
          </div>
          <div className="text-2xl font-bold">€{(portfolio?.balance ?? 0).toFixed(2)}</div>
        </div>
      </div>

      {/* Holdings Table */}
      <div className="card p-6">
        <h2 className="font-bold text-lg mb-4">Holdings</h2>
        {loading && <p className="text-gray-400">Laden...</p>}
        {holdings.length === 0 && !loading && (
          <div className="text-center text-gray-400 py-8">
            Geen holdings. Ga naar Tokens om te beginnen met traden.
          </div>
        )}
        {holdings.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="text-gray-400 text-sm border-b border-white/10">
                  <th className="pb-3">Token</th>
                  <th className="pb-3 text-right">Amount</th>
                  <th className="pb-3 text-right">Avg Buy</th>
                  <th className="pb-3 text-right">Current</th>
                  <th className="pb-3 text-right">Value</th>
                  <th className="pb-3 text-right">P&L</th>
                  <th className="pb-3"></th>
                </tr>
              </thead>
              <tbody>
                {holdings.map((tb) => {
                  const tokenPnl = (tb.token.price - tb.avg_price) * tb.balance;
                  return (
                    <tr key={tb.token.address} className="border-t border-white/5 hover:bg-white/5">
                      <td className="py-3">
                        <div className="font-semibold">{tb.token.symbol}</div>
                        <div className="text-sm text-gray-400">{tb.token.name}</div>
                      </td>
                      <td className="py-3 text-right">{tb.balance.toFixed(4)}</td>
                      <td className="py-3 text-right">€{tb.avg_price.toFixed(6)}</td>
                      <td className="py-3 text-right">€{tb.token.price.toFixed(6)}</td>
                      <td className="py-3 text-right">€{tb.value.toFixed(2)}</td>
                      <td className={`py-3 text-right ${tokenPnl >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                        {tokenPnl >= 0 ? '+' : ''}€{Math.abs(tokenPnl).toFixed(2)}
                      </td>
                      <td className="py-3 text-right">
                        <button
                          onClick={() => {
                            setSellToken({
                              address: tb.token.address,
                              symbol: tb.token.symbol,
                              name: tb.token.name,
                              price: tb.token.price,
                              balance: tb.balance,
                            });
                            setSellAmount(tb.balance);
                          }}
                          className="px-3 py-1 bg-red-500/20 text-red-400 rounded-lg text-sm hover:bg-red-500/30"
                        >
                          Verkopen
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Sell Modal */}
      {sellToken && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={() => setSellToken(null)}>
          <div className="card p-6 max-w-md w-full" onClick={(e) => e.stopPropagation()}>
            <h2 className="text-xl font-bold mb-4">Verkopen {sellToken.symbol}</h2>
            <div className="mb-4 text-sm text-gray-400">
              <div>Prijs: €{sellToken.price.toFixed(6)}</div>
              <div>Balans: {sellToken.balance.toFixed(4)} tokens</div>
            </div>
            <div className="mb-4">
              <label className="block text-sm text-gray-400 mb-1">Aantal</label>
              <input
                type="number"
                value={sellAmount}
                onChange={(e) => setSellAmount(Number(e.target.value))}
                max={sellToken.balance}
                className="input-field"
              />
              <div className="text-sm text-gray-400 mt-1">
                Opbrengst: €{(sellAmount * sellToken.price).toFixed(2)}
              </div>
            </div>
            <div className="flex gap-3">
              <button onClick={() => setSellToken(null)} className="btn-secondary flex-1">Annuleren</button>
              <button onClick={handleSell} className="btn-primary flex-1 bg-red-500 !from-red-500 !to-red-600">
                Verkopen
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
