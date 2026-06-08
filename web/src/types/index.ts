export interface Token {
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  price: number;
  price_change_24h: number;
  market_cap: number;
  volume_24h: number;
  liquidity: number;
  security_score: number;
  discovered_at: string;
  tags: string[];
}

export interface Trade {
  id: string;
  token_address: string;
  token_symbol: string;
  type: 'buy' | 'sell';
  amount_in: number;
  amount_out: number;
  price: number;
  gas_used: number;
  gas_price: number;
  tx_hash: string;
  status: string;
  profit_loss: number;
  executed_at: string;
  is_paper_trade: boolean;
}

export interface TokenBalance {
  token: Token;
  balance: number;
  value: number;
  avg_price: number;
}

export interface Portfolio {
  id: string;
  balance: number;
  currency: string;
  eth_balance: number;
  token_balances: Record<string, TokenBalance>;
  total_value: number;
  profit_loss: number;
  profit_loss_pct: number;
  updated_at: string;
}

export interface BotStatus {
  is_running: boolean;
  mode: string;
  started_at?: string;
  total_trades: number;
  profitable_trades: number;
  total_profit_loss: number;
  current_balance: number;
  active_opportunities: number;
}

export interface Metrics {
  uptime: number;
  total_trades: number;
  successful_trades: number;
  failed_trades: number;
  total_volume: number;
  total_profit_loss: number;
  win_rate: number;
  average_profit_per_trade: number;
  tokens_discovered: number;
  opportunities_found: number;
}

export interface Config {
  paper_trading_mode: boolean;
  initial_balance: number;
  trade_amount: number;
  max_slippage: number;
  min_liquidity: number;
  max_price_impact: number;
  scan_interval_sec: number;
  gas_limit: number;
  max_gas_price: number;
  environment: string;
  eth_node_url?: string;
  stop_loss?: number;
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}
