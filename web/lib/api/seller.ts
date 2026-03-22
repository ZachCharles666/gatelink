import { apiClient } from "@/lib/api/client";

export type SellerAuthResponse = {
  token: string;
  seller_id: string;
};

export type SellerAccount = {
  id: string;
  seller_id: string;
  vendor: string;
  status: string;
  health_score: number;
  expected_rate: number;
  authorized_credits_usd: number;
  consumed_credits_usd: number;
  total_credits_usd: number;
  expire_at: string;
  created_at: string;
  updated_at: string;
};

export type SellerAccountsResponse = {
  accounts: SellerAccount[];
};

export type SellerSettlement = {
  id: string;
  seller_id: string;
  amount_usd: number;
  status: string;
  period_start: string;
  period_end: string;
  created_at: string;
  paid_at?: string;
  tx_hash?: string;
};

export type SellerEarningsResponse = {
  pending_usd: number;
  total_earned_usd: number;
  settlements: SellerSettlement[];
};

export type SellerSettlementsResponse = {
  total: number;
  settlements: SellerSettlement[];
};

export type AddAccountRequest = {
  vendor: string;
  api_key: string;
  authorized_credits_usd: number;
  expected_rate: number;
  expire_at: string;
  total_credits_usd?: number;
};

export type AddAccountResponse = {
  account_id: string;
  api_key_hint?: string;
  status: string;
  message: string;
};

export type RequestSettlementResponse = {
  message: string;
};

export const sellerAPI = {
  login(phone: string, code: string) {
    return apiClient.post<SellerAuthResponse>("/api/v1/seller/auth/login", {
      phone,
      code,
    });
  },
  register(phone: string, code: string, displayName?: string) {
    return apiClient.post<SellerAuthResponse>("/api/v1/seller/auth/register", {
      phone,
      code,
      display_name: displayName || "",
    });
  },
  getAccounts() {
    return apiClient.get<SellerAccountsResponse>("/api/v1/seller/accounts");
  },
  addAccount(data: AddAccountRequest) {
    return apiClient.post<AddAccountResponse>("/api/v1/seller/accounts", data);
  },
  getAccountUsage(id: string) {
    return apiClient.get(`/api/v1/seller/accounts/${id}/usage`);
  },
  updateAuthorization(id: string, authorizedCreditsUSD: number) {
    return apiClient.patch(`/api/v1/seller/accounts/${id}/authorization`, {
      authorized_credits_usd: authorizedCreditsUSD,
    });
  },
  revokeAuthorization(id: string) {
    return apiClient.delete(`/api/v1/seller/accounts/${id}/authorization`);
  },
  getEarnings() {
    return apiClient.get<SellerEarningsResponse>("/api/v1/seller/earnings");
  },
  getSettlements(page = 1) {
    return apiClient.get<SellerSettlementsResponse>(`/api/v1/seller/settlements?page=${page}`);
  },
  requestSettlement() {
    return apiClient.post<RequestSettlementResponse>("/api/v1/seller/settlements/request");
  },
};
