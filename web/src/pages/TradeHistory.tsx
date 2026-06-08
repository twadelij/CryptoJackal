import { ArrowDown, ArrowUp } from 'lucide-react';
import { useFetch } from '../hooks/useApi';
import * as api from '../lib/api';
import type { Trade } from '../types';

export default function TradeHistory() {
  const { data: historyData, loading } = useFetch(
    () => api.getPaperHistory().then(r => r.data as Trade[]),
    10000
  );

  const trades = historyData ?? [];

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Trade History</h1>

      {loading && <p className="text-gray-400">Laden...</p>}

      {trades.length === 0 && !loading && (
        <div className="card p-6 text-center text-gray-400">
          Nog geen trades. Ga naar Tokens om te beginnen.
        </div>
      )}

      <div className="space-y-3">
        {trades.map((trade) => (
          <div key={trade.id} className="card p-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className={`w-10 h-10 rounded-full flex items-center justify-center font-bold ${
                trade.type === 'buy'
                  ? 'bg-green-500/20 text-green-400'
                  : 'bg-red-500/20 text-red-400'
              }`}>
                {trade.type === 'buy' ? <ArrowDown size={18} /> : <ArrowUp size={18} />}
              </div>
              <div>
                <div className="font-semibold">
                  {trade.type.toUpperCase()} {trade.token_symbol}
                </div>
                <div className="text-sm text-gray-400">
                  {new Date(trade.executed_at).toLocaleString()}
                </div>
                <div className={`text-sm ${trade.status === 'executed' ? 'text-green-400' : 'text-red-400'}`}>
                  {trade.status}
                </div>
              </div>
            </div>
            <div className="text-right">
              <div className="font-semibold">
                €{(trade.amount_in * trade.price).toFixed(2)}
              </div>
              <div className="text-sm text-gray-400">
                {trade.amount_in.toFixed(2)} tokens @ €{trade.price.toFixed(6)}
              </div>
              {trade.profit_loss !== 0 && (
                <div className={`text-sm ${trade.profit_loss >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                  P&L: {trade.profit_loss >= 0 ? '+' : ''}€{Math.abs(trade.profit_loss).toFixed(2)}
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
