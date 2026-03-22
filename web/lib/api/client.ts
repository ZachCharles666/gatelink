import axios, { AxiosError, type AxiosRequestConfig, type AxiosResponse } from "axios";

type APIEnvelope<T> = {
  code: number;
  msg: string;
  data: T;
};

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const SELLER_TOKEN_KEY = "seller_token";

const http = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
});

http.interceptors.request.use((config) => {
  if (typeof window === "undefined") {
    return config;
  }

  const token = window.localStorage.getItem(SELLER_TOKEN_KEY);
  if (token) {
    config.headers = config.headers ?? {};
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(undefined, (error: AxiosError<APIEnvelope<unknown>>) => {
  if (error.response?.status === 401 && typeof window !== "undefined") {
    window.localStorage.removeItem(SELLER_TOKEN_KEY);
    window.location.assign("/seller/login");
  }

  const message =
    error.response?.data?.msg ||
    error.message ||
    "request failed";

  return Promise.reject(new Error(message));
});

function unwrap<T>(request: Promise<AxiosResponse<APIEnvelope<T>>>) {
  return request.then((response) => {
    const body = response.data;
    if (body.code !== 0) {
      throw new Error(body.msg || "unknown error");
    }
    return body.data;
  });
}

export const authStorage = {
  getToken() {
    if (typeof window === "undefined") {
      return null;
    }
    return window.localStorage.getItem(SELLER_TOKEN_KEY);
  },
  setToken(token: string) {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(SELLER_TOKEN_KEY, token);
    }
  },
  clearToken() {
    if (typeof window !== "undefined") {
      window.localStorage.removeItem(SELLER_TOKEN_KEY);
    }
  },
};

export const apiClient = {
  get<T>(url: string, config?: AxiosRequestConfig) {
    return unwrap<T>(http.get<APIEnvelope<T>>(url, config));
  },
  post<T>(url: string, data?: unknown, config?: AxiosRequestConfig) {
    return unwrap<T>(http.post<APIEnvelope<T>>(url, data, config));
  },
  patch<T>(url: string, data?: unknown, config?: AxiosRequestConfig) {
    return unwrap<T>(http.patch<APIEnvelope<T>>(url, data, config));
  },
  delete<T>(url: string, config?: AxiosRequestConfig) {
    return unwrap<T>(http.delete<APIEnvelope<T>>(url, config));
  },
};
