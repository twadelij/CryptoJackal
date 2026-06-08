import { BookOpen, Shield, TrendingUp, Wallet, Settings, AlertTriangle } from 'lucide-react';

export default function Help() {
  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Help</h1>

      <section className="card p-6 mb-6">
        <div className="flex items-center gap-3 mb-4">
          <BookOpen size={22} className="text-jackal-pink" />
          <h2 className="text-lg font-semibold">Getting Started</h2>
        </div>
        <p className="text-gray-300 mb-4">
          CryptoJackal is a trading bot that can operate in two modes:
        </p>
        <ul className="list-disc list-inside text-gray-300 space-y-2">
          <li><strong>Paper Trading (default)</strong> - Practice with fake money. No setup needed.</li>
          <li><strong>Live Trading</strong> - Real trades with real money. Requires an Ethereum node and wallet.</li>
        </ul>
      </section>

      <section className="card p-6 mb-6">
        <div className="flex items-center gap-3 mb-4">
          <TrendingUp size={22} className="text-jackal-pink" />
          <h2 className="text-lg font-semibold">Dashboard</h2>
        </div>
        <p className="text-gray-300 mb-2">
          The main overview page. Here you can:
        </p>
        <ul className="list-disc list-inside text-gray-300 space-y-1">
          <li>See your current balance and profit/loss</li>
          <li>View total trades and win rate</li>
          <li>Start or stop the auto-trading bot</li>
          <li>Reset your paper trading portfolio to the starting balance</li>
        </ul>
      </section>

      <section className="card p-6 mb-6">
        <div className="flex items-center gap-3 mb-4">
          <Wallet size={22} className="text-jackal-pink" />
          <h2 className="text-lg font-semibold">Tokens & Portfolio</h2>
        </div>
        <p className="text-gray-300 mb-2">
          <strong>Tokens</strong> - Browse trending and newly listed tokens. Search by name or symbol. Click any token to buy it.
        </p>
        <p className="text-gray-300 mb-2">
          <strong>Portfolio</strong> - See all tokens you own, their current value, and profit/loss per token. Sell tokens here.
        </p>
        <p className="text-gray-300">
          <strong>History</strong> - A complete log of all your trades with timestamps and prices.
        </p>
      </section>

      <section className="card p-6 mb-6">
        <div className="flex items-center gap-3 mb-4">
          <Settings size={22} className="text-jackal-pink" />
          <h2 className="text-lg font-semibold">Settings Explained</h2>
        </div>
        <div className="space-y-4 text-gray-300">
          <div>
            <div className="font-semibold text-white">Trading Mode</div>
            <div className="text-sm">Paper = fake money. Live = real money on the Ethereum blockchain.</div>
          </div>
          <div>
            <div className="font-semibold text-white">Start Balance</div>
            <div className="text-sm">Your initial amount in paper trading mode (default: 10,000 EUR).</div>
          </div>
          <div>
            <div className="font-semibold text-white">Max Trade Size</div>
            <div className="text-sm">Maximum amount spent on a single trade (default: 100 EUR).</div>
          </div>
          <div>
            <div className="font-semibold text-white">Stop Loss %</div>
            <div className="text-sm">If a token drops this percentage, sell it automatically (default: 5%).</div>
          </div>
          <div>
            <div className="font-semibold text-white">Max Slippage</div>
            <div className="text-sm">Maximum price difference allowed between quote and execution (default: 0.5%).</div>
          </div>
          <div>
            <div className="font-semibold text-white">Ethereum Node URL</div>
            <div className="text-sm">Your RPC endpoint (Infura, Alchemy, QuickNode). Required for live trading.</div>
          </div>
          <div>
            <div className="font-semibold text-white">Private Key</div>
            <div className="text-sm">Your wallet private key. Only needed for live trading. Never share this.</div>
          </div>
        </div>
      </section>

      <section className="card p-6 mb-6">
        <div className="flex items-center gap-3 mb-4">
          <Shield size={22} className="text-jackal-pink" />
          <h2 className="text-lg font-semibold">Security Tips</h2>
        </div>
        <ul className="list-disc list-inside text-gray-300 space-y-2">
          <li>Always start with <strong>Paper Trading</strong> to learn how the bot works.</li>
          <li>Never share your private key or API keys.</li>
          <li>Store your private key in environment variables, not in code.</li>
          <li>Use a dedicated wallet with limited funds for live trading.</li>
          <li>Enable 2FA on all exchange and node provider accounts.</li>
        </ul>
      </section>

      <section className="card p-6 border border-yellow-500/30">
        <div className="flex items-center gap-3 mb-4">
          <AlertTriangle size={22} className="text-yellow-400" />
          <h2 className="text-lg font-semibold text-yellow-400">Risk Warning</h2>
        </div>
        <p className="text-gray-300">
          Cryptocurrency trading involves significant risk. You can lose all your money.
          This software is provided as-is with no guarantees. Always use paper trading first.
          Never trade with money you cannot afford to lose.
        </p>
      </section>
    </div>
  );
}
