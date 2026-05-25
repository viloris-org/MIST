export interface SessionPool {
  sessions: number;
  idle_sessions: number;
  open_streams: number;
  cumulative_sessions: number;
  cumulative_streams: number;
}

export interface ClientStats {
  session_pool: SessionPool;
}

export interface TUNStats {
  name: string;
  mtu: number;
  address: string;
  is_up: boolean;
  rx_bytes: number;
  tx_bytes: number;
  rx_packets: number;
  tx_packets: number;
}

export interface DNSStats {
  queries_forwarded: number;
  queries_cached: number;
  queries_failed: number;
  cache_size: number;
  upstreams: string[];
}

export interface StatusResponse {
  version: string;
  commit: string;
  date: string;
  server: string;
  server_state: 'running' | 'error' | 'starting' | string;
  started_at: string;
  updated_at: string;
  active_connections: number;
  accepted_connections: number;
  last_error?: string;
  client?: ClientStats;
  tun?: TUNStats;
  dns?: DNSStats;
}
