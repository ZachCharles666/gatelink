import { apiClient } from "@/lib/api/client";

export type BuyerAuthResponse = {
  token: string;
  buyer_id: string;
  api_key?: string;
};

export type BuyerBalanceResponse = {
  balance_usd: number;
  total_consumed_usd: number;
  tier: string;
};

export type BuyerTopupRecord = {
  id: string;
  buyer_id: string;
  amount_usd: number;
  tx_hash: string;
  network: string;
  status: string;
  confirmed_at?: string;
  rejected_at?: string;
  notes?: string;
  created_at: string;
};

export type BuyerTopupListResponse = {
  total: number;
  records: BuyerTopupRecord[];
};

export type BuyerTopupResponse = {
  topup_id: string;
  amount_usd: number;
  network: string;
  status: string;
  message: string;
};

export type BuyerUsageRecord = {
  vendor: string;
  model: string;
  input_tokens: number;
  output_tokens: number;
  cost_usd: number;
  buyer_charged_usd: number;
  created_at: string;
};

export type BuyerUsageResponse = {
  total: number;
  records: BuyerUsageRecord[];
};

export type BuyerResetAPIKeyResponse = {
  api_key: string;
  message: string;
};

const BUYER_TOKEN_KEY = "buyer_token";
const BUYER_API_KEY = "buyer_api_key";

export const buyerStorage = {
  getToken() {
    if (typeof window === "undefined") {
      return null;
    }
    return window.localStorage.getItem(BUYER_TOKEN_KEY);
  },
  setToken(token: string) {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(BUYER_TOKEN_KEY, token);
    }
  },
  clearToken() {
    if (typeof window !== "undefined") {
      window.localStorage.removeItem(BUYER_TOKEN_KEY);
    }
  },
  getAPIKey() {
    if (typeof window === "undefined") {
      return null;
    }
    return window.localStorage.getItem(BUYER_API_KEY);
  },
  setAPIKey(apiKey: string) {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(BUYER_API_KEY, apiKey);
    }
  },
  clearAPIKey() {
    if (typeof window !== "undefined") {
      window.localStorage.removeItem(BUYER_API_KEY);
    }
  },
};

export const buyerAPI = {
  register(data: {
    email?: string;
    phone?: string;
    password: string;
  }) {
    return apiClient.post<BuyerAuthResponse>("/api/v1/buyer/auth/register", data);
  },
  login(data: {
    email?: string;
    phone?: string;
    password: string;
  }) {
    return apiClient.post<BuyerAuthResponse>("/api/v1/buyer/auth/login", data);
  },
  getBalance() {
    return apiClient.get<BuyerBalanceResponse>("/api/v1/buyer/balance");
  },
  submitTopup(data: {
    amount_usd: number;
    tx_hash: string;
    network: "TRC20" | "ERC20";
  }) {
    return apiClient.post<BuyerTopupResponse>("/api/v1/buyer/topup", data);
  },
  getTopupRecords(page = 1) {
    return apiClient.get<BuyerTopupListResponse>(`/api/v1/buyer/topup/records?page=${page}`);
  },
  getUsage(page = 1) {
    return apiClient.get<BuyerUsageResponse>(`/api/v1/buyer/usage?page=${page}`);
  },
  resetAPIKey() {
    return apiClient.post<BuyerResetAPIKeyResponse>("/api/v1/buyer/apikeys/reset");
  },
};
