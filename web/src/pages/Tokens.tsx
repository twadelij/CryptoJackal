import { useState } from 'react';
import { Search, TrendingUp, TrendingDown } from 'lucide-react';
import { useFetch } from '../hooks/useApi';
import * as api from '../lib/api';
import { Toast } from '../components/Toast';
import type { Token } from '../types';

export default function Tokens() {
  const [activeTab, setActiveTab] = useState<'trending' | 'new'>('new');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedToken, setSelectedToken] = useState<Token | null>(null);
  const [tradeAmount, setTradeAmount] = useState(1000);
  const [tradeType, setTradeType] = useState<'buy' | 'sell'>('buy');
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  const { data: trendingData, loading: trendingLoading } = useFetch(
    () => api.getTrendingTokens().then(r => r.data as Token[]),
    60000
  );
  const { data: newData, loading: newLoading } = useFetch(
    () => api.getNewTokens().then(r => r.data as Token[]),
    60000
  );

  const tokens = activeTab === 'trending' ? (trendingData ?? []) : (newData ?? []);
  const filtered = tokens.filter(t =>
    t.symbol.toLowerCase().includes(searchQuery.toLowerCase()) ||
    t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    t.address.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const showToast = (message: string, type: 'success' | 'error') => {
    setToast({ message, type });
  };

  const handleTrade = async () => {
    if (!selectedToken) return;
    try {
      await api.executePaperTrade(
        selectedToken.address,
        selectedToken.symbol,
        selectedToken.name,
        selectedToken.price,
        tradeAmount,
        tradeType
      );
      showToast(`${tradeType.toUpperCase()} order uitgevoerd voor ${selectedToken.symbol}`, 'success');
      setSelectedToken(null);
    } catch (e) {
      showToast('Trade mislukt: ' + (e instanceof Error ? e.message : ''), 'error');
    }
  };

  return (
    <div>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      <h1 className="text-2xl font-bold mb-6">Token Discovery</h1>

      {/* Search & Tabs */}
      <div className="flex flex-col sm:flex-row gap-4 mb-6">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" size={18} />
          <input
            type="text"
            placeholder="Zoek tokens..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="input-field pl-10"
          />
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setActiveTab('new')}
            className={`px-4 py-2 rounded-lg font-semibold ${activeTab === 'new' ? 'bg-jackal-pink' : 'bg-white/10'}`}
          >
            New
          </button>
          <button
            onClick={() => setActiveTab('trending')}
            className={`px-4 py-2 rounded-lg font-semibold ${activeTab === 'trending' ? 'bg-jackal-pink' : 'bg-white/10'}`}
          >
            Trending
          </button>
        </div>
      </div>

      {/* Token List */}
      {(trendingLoading || newLoading) && <p className="text-gray-400">Laden...</p>}

      <div className="space-y-3">
        {filtered.length === 0 && !trendingLoading && !newLoading && (
          <div className="card p-6 text-center text-gray-400">
            Geen tokens gevonden. Probeer een andere zoekterm.
          </div>
        )}

        {filtered.slice(0, 30).map((token) => (
          <div
            key={token.address}
            onClick={() => { setSelectedToken(token); setTradeType('buy'); }}
            className="card p-4 flex items-center justify-between cursor-pointer hover:bg-white/10 transition-colors"
          >
            <div className="flex items-center gap-4">
              <div className="w-10 h-10 rounded-full bg-jackal-pink/20 flex items-center justify-center font-bold text-jackal-pink">
                {token.symbol.slice(0, 2)}
              </div>
              <div>
                <div className="font-semibold">{token.symbol}</div>
                <div className="text-sm text-gray-400">{token.name}</div>
                <div className="text-xs text-gray-500">{token.address.slice(0, 6)}...{token.address.slice(-4)}</div>
              </div>
            </div>
            <div className="text-right">
              <div className="font-semibold">€{token.price < 0.01 ? token.price.toExponential(2) : token.price.toFixed(6)}</div>
              <div className={`text-sm flex items-center justify-end gap-1 ${token.price_change_24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                {token.price_change_24h >= 0 ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                {token.price_change_24h >= 0 ? '+' : ''}{token.price_change_24h.toFixed(2)}%
              </div>
              <div className="text-xs text-gray-500">Vol: €{(token.volume_24h / 1000).toFixed(1)}K</div>
            </div>
          </div>
        ))}
      </div>

      {/* Trade Modal */}
      {selectedToken && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={() => setSelectedToken(null)}>
          <div className="card p-6 max-w-md w-full" onClick={(e) => e.stopPropagation()}>
            <h2 className="text-xl font-bold mb-4">Trade {selectedToken.symbol}</h2>
            <div className="mb-4 text-sm text-gray-400">
              <div>Naam: {selectedToken.name}</div>
              <div>Prijs: €{selectedToken.price < 0.01 ? selectedToken.price.toExponential(2) : selectedToken.price.toFixed(6)}</div>
              <div>Security Score: {(selectedToken.security_score * 100).toFixed(0)}%</div>
            </div>

            <div className="flex gap-2 mb-4">
              <button
                onClick={() => setTradeType('buy')}
                className={`flex-1 py-2 rounded-lg font-semibold ${tradeType === 'buy' ? 'bg-green-500/20 text-green-400 border border-green-500/50' : 'bg-white/5'}`}
              >
                Kopen
              </button>
              <button
                onClick={() => setTradeType('sell')}
                className={`flex-1 py-2 rounded-lg font-semibold ${tradeType === 'sell' ? 'bg-red-500/20 text-red-400 border border-red-500/50' : 'bg-white/5'}`}
              >
                Verkopen
              </button>
            </div>

            <div className="mb-4">
              <label className="block text-sm text-gray-400 mb-1">Aantal tokens</label>
              <input
                type="number"
                value={tradeAmount}
                onChange={(e) => setTradeAmount(Number(e.target.value))}
                className="input-field"
              />
              <div className="text-sm text-gray-400 mt-1">
                Kosten: €{(tradeAmount * selectedToken.price).toFixed(2)}
              </div>
            </div>

            <div className="flex gap-3">
              <button onClick={() => setSelectedToken(null)} className="btn-secondary flex-1">
                Annuleren
              </button>
              <button onClick={handleTrade} className="btn-primary flex-1">
                {tradeType === 'buy' ? 'Kopen' : 'Verkopen'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
