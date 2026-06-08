import { useState } from 'react';
import { ArrowRight, ArrowLeft, CheckCircle } from 'lucide-react';
import * as api from '../lib/api';
import { Toast } from '../components/Toast';

export default function SetupWizard() {
  const [step, setStep] = useState(1);
  const [mode, setMode] = useState<'paper' | 'live'>('paper');
  const [ethNode, setEthNode] = useState('');
  const [initialBalance, setInitialBalance] = useState(10000);
  const [tradeAmount, setTradeAmount] = useState(100);
  const [stopLoss, setStopLoss] = useState(5);
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  const showToast = (message: string, type: 'success' | 'error') => {
    setToast({ message, type });
  };

  const handleSave = async () => {
    setLoading(true);
    try {
      await api.updateConfig({
        paper_trading_mode: mode === 'paper',
        initial_balance: initialBalance,
        eth_node_url: ethNode,
        trade_amount: tradeAmount,
        max_slippage: 0.5,
        stop_loss: stopLoss,
      });
      showToast('Config opgeslagen!', 'success');
    } catch (e) {
      showToast('Config opslaan mislukt: ' + (e instanceof Error ? e.message : ''), 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto">
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      <h1 className="text-2xl font-bold mb-6">Setup Wizard</h1>

      {/* Step indicators */}
      <div className="flex justify-between mb-8">
        {[1, 2, 3].map((s) => (
          <div key={s} className="flex items-center gap-2">
            <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold ${
              s <= step ? 'bg-jackal-pink' : 'bg-gray-600'
            }`}>
              {s}
            </div>
            <span className={`text-sm hidden sm:block ${s <= step ? 'text-white' : 'text-gray-500'}`}>
              {s === 1 ? 'Mode' : s === 2 ? 'API Keys' : 'Trading'}
            </span>
          </div>
        ))}
      </div>

      {/* Step 1: Mode */}
      {step === 1 && (
        <div className="card p-6">
          <h2 className="text-lg font-semibold mb-4">Selecteer Trading Mode</h2>
          <div className="grid grid-cols-2 gap-4 mb-6">
            <button
              onClick={() => setMode('paper')}
              className={`card p-6 text-left transition-all ${mode === 'paper' ? 'border-jackal-pink' : ''}`}
            >
              <div className="text-3xl mb-2">📝</div>
              <div className="font-bold">Paper Trading</div>
              <div className="text-sm text-gray-400">Oefen met virtueel geld</div>
            </button>
            <button
              onClick={() => setMode('live')}
              className={`card p-6 text-left transition-all ${mode === 'live' ? 'border-jackal-pink' : ''}`}
            >
              <div className="text-3xl mb-2">💰</div>
              <div className="font-bold">Live Trading</div>
              <div className="text-sm text-gray-400">Trade met echt geld</div>
            </button>
          </div>
          <button onClick={() => setStep(2)} className="btn-primary w-full flex items-center justify-center gap-2">
            Verder <ArrowRight size={18} />
          </button>
        </div>
      )}

      {/* Step 2: API Keys */}
      {step === 2 && (
        <div className="card p-6">
          <h2 className="text-lg font-semibold mb-4">API Configuratie</h2>
          <div className="space-y-4 mb-6">
            <div>
              <label className="block text-sm text-gray-400 mb-1">Ethereum Node URL (optioneel voor paper)</label>
              <input
                type="text"
                value={ethNode}
                onChange={(e) => setEthNode(e.target.value)}
                placeholder="https://mainnet.infura.io/v3/YOUR_KEY"
                className="input-field"
              />
            </div>
            {mode === 'live' && (
              <div>
                <label className="block text-sm text-gray-400 mb-1">Wallet Private Key (alleen live)</label>
                <input
                  type="password"
                  placeholder="0x..."
                  className="input-field"
                />
                <p className="text-xs text-yellow-400 mt-1">⚠️ Deel nooit je private key</p>
              </div>
            )}
          </div>
          <div className="flex gap-4">
            <button onClick={() => setStep(1)} className="btn-secondary flex-1 flex items-center justify-center gap-2">
              <ArrowLeft size={18} /> Terug
            </button>
            <button onClick={() => setStep(3)} className="btn-primary flex-1 flex items-center justify-center gap-2">
              Verder <ArrowRight size={18} />
            </button>
          </div>
        </div>
      )}

      {/* Step 3: Trading Settings */}
      {step === 3 && (
        <div className="card p-6">
          <h2 className="text-lg font-semibold mb-4">Trading Instellingen</h2>
          <div className="space-y-4 mb-6">
            <div>
              <label className="block text-sm text-gray-400 mb-1">Start Balans (EUR)</label>
              <input
                type="number"
                value={initialBalance}
                onChange={(e) => setInitialBalance(Number(e.target.value))}
                className="input-field"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">Max Trade Grootte (EUR)</label>
              <input
                type="number"
                value={tradeAmount}
                onChange={(e) => setTradeAmount(Number(e.target.value))}
                className="input-field"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">Stop Loss %</label>
              <input
                type="number"
                value={stopLoss}
                onChange={(e) => setStopLoss(Number(e.target.value))}
                className="input-field"
              />
            </div>
          </div>
          <div className="flex gap-4">
            <button onClick={() => setStep(2)} className="btn-secondary flex-1 flex items-center justify-center gap-2">
              <ArrowLeft size={18} /> Terug
            </button>
            <button
              onClick={handleSave}
              disabled={loading}
              className="btn-primary flex-1 flex items-center justify-center gap-2 disabled:opacity-50"
            >
              {loading ? 'Bezig...' : <><CheckCircle size={18} /> Opslaan</>}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
