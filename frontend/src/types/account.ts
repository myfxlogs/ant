export interface Account {
  id: string;
  userId?: string;
  mtType: 'MT4' | 'MT5';
  brokerCompany: string;
  brokerServer: string;
  brokerHost: string;
  login: string;
  alias?: string;
  balance?: number;
  credit?: number;
  equity?: number;
  margin?: number;
  freeMargin?: number;
  marginLevel?: number;
  profit?: number;
  profitPercent?: number;
  leverage?: number;
  currency?: string;
  status: string;
  accountType?: 'demo' | 'real' | 'contest' | 'unknown';
  isInvestor?: boolean;
  isDisabled: boolean;
  lastError?: string;
  token?: string;
  connectedAt?: Date;
  createdAt?: Date;
  updatedAt?: Date;
  streamStatus?: string;
  accountStatus?: string;
  lastConnectedAt?: Date;
  lastCheckedAt?: Date;
  type?: string;
  method?: number;
  isPublic?: boolean;
}

export interface BindAccountRequest {
  mtType: 'MT4' | 'MT5';
  brokerCompany: string;
  brokerServer: string;
  brokerHost: string;
  login: string;
  password: string;
  alias?: string;
}
