import type { ApiResponse, BotStatus, Config, Metrics, Portfolio, Token, Trade } from '../types';

const API_BASE = '/api';

async function fetchJson<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, options);
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${res.statusText}`);
  }
  return res.json();
}

export async function getHealth(): Promise<ApiResponse<{ status: string; version: string }>> {
  return fetchJson(`${API_BASE}/health`);
}

export async function getConfig(): Promise<ApiResponse<Config>> {
  return fetchJson(`${API_BASE}/config`);
}

export async function updateConfig(config: Partial<Config>): Promise<ApiResponse<string>> {
  return fetchJson(`${API_BASE}/config`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
}

export async function getBotStatus(): Promise<ApiResponse<BotStatus>> {
  return fetchJson(`${API_BASE}/bot/status`);
}

export async function startBot(): Promise<ApiResponse<string>> {
  return fetchJson(`${API_BASE}/bot/start`, { method: 'POST' });
}

export async function stopBot(): Promise<ApiResponse<string>> {
  return fetchJson(`${API_BASE}/bot/stop`, { method: 'POST' });
}

export async function getPaperBalance(): Promise<ApiResponse<Portfolio>> {
  return fetchJson(`${API_BASE}/paper/balance`);
}

export async function resetPaperBalance(): Promise<ApiResponse<string>> {
  return fetchJson(`${API_BASE}/paper/reset`, { method: 'POST' });
}

export async function executePaperTrade(
  tokenAddress: string,
  tokenSymbol: string,
  tokenName: string,
  price: number,
  amount: number,
  type: 'buy' | 'sell'
): Promise<ApiResponse<Trade>> {
  return fetchJson(`${API_BASE}/paper/trade`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      token_address: tokenAddress,
      token_symbol: tokenSymbol,
      token_name: tokenName,
      price,
      amount,
      type,
    }),
  });
}

export async function getPaperHistory(): Promise<ApiResponse<Trade[]>> {
  return fetchJson(`${API_BASE}/paper/history`);
}

export async function getTrendingTokens(): Promise<ApiResponse<Token[]>> {
  return fetchJson(`${API_BASE}/discovery/trending`);
}

export async function getNewTokens(chain = 'ethereum'): Promise<ApiResponse<Token[]>> {
  return fetchJson(`${API_BASE}/discovery/new?chain=${chain}`);
}

export async function analyzeToken(address: string): Promise<ApiResponse<Token>> {
  return fetchJson(`${API_BASE}/discovery/analyze/${address}`);
}

export async function getMetrics(): Promise<ApiResponse<Metrics>> {
  return fetchJson(`${API_BASE}/metrics`);
}
