export interface SessionPool {
  sessions: number;
  idle_sessions: number;
  open_streams: number;
  cumulative_sessions: number;
  cumulative_streams: number;
}

export interface SessionInfo {
  seq: number;
  stream_count: number;
  packet_count: number;
  age_ms: number;
  is_idle: boolean;
  is_closed: boolean;
}

export interface ClientStats {
  session_pool: SessionPool;
  sessions: SessionInfo[];
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

export interface RuntimeConfig {
  server: string;
  server_state: string;
  last_error?: string;
  listen: string;
  inbound: string;
  redirect_listen: string;
  min_idle_session: number;
  tls_min_version: string;
  insecure: boolean;
  tun?: {
    name: string;
    mtu: number;
    address: string;
  };
  dns?: {
    listen: string;
    upstream: string;
  };
  web?: {
    listen: string;
    has_password: boolean;
  };
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
