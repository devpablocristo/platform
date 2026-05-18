import axios, {
  AxiosError,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
  type AxiosInstance,
} from "axios";
import { dispatchAuthForceLogout } from "../browser/events";

declare module "axios" {
  export interface InternalAxiosRequestConfig {
    _retry?: boolean;
  }
}

export interface AxiosTokenStorage {
  getAccessToken(): string | null;
  getRefreshToken(): string | null;
  setAccessToken(token: string): void;
  setRefreshToken(token: string): void;
  clear(): void;
}

export interface RefreshTokenPair {
  accessToken: string;
  refreshToken?: string | null;
}

export interface RefreshRequestConfig {
  path: string;
  method?: "GET" | "POST";
  useRefreshToken?: boolean;
  body?: unknown;
  mapResponse(data: unknown): RefreshTokenPair;
}

export interface AuthenticatedAxiosClientOptions {
  baseURL?: string;
  timeoutMs?: number;
  tokenStorage: AxiosTokenStorage;
  refreshRequest: RefreshRequestConfig;
  invalidTokenMatcher?: (error: unknown) => boolean;
}

type QueueEntry = {
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
};

function defaultInvalidTokenMatcher(error: unknown): boolean {
  const axiosError = error as AxiosError;
  const status = axiosError.response?.status;
  if (status !== 401 && status !== 403) {
    return false;
  }

  const data = axiosError.response?.data as unknown;
  const haystack =
    typeof data === "string"
      ? data
      : data && typeof data === "object"
        ? JSON.stringify(data)
        : "";
  const message = haystack.toLowerCase();
  return (
    message.includes("invalid token") ||
    message.includes("token inval") ||
    message.includes("token invál") ||
    message.includes("jwt") ||
    message.includes("signature") ||
    message.includes("expired")
  );
}

export class AuthenticatedAxiosClient {
  private readonly client: AxiosInstance;

  private readonly tokenStorage: AxiosTokenStorage;

  private readonly refreshRequest: Required<Pick<RefreshRequestConfig, "path" | "method" | "useRefreshToken" | "mapResponse">> &
    Pick<RefreshRequestConfig, "body">;

  private readonly invalidTokenMatcher: (error: unknown) => boolean;

  private isRefreshing = false;

  private failedQueue: QueueEntry[] = [];

  constructor(options: AuthenticatedAxiosClientOptions) {
    this.tokenStorage = options.tokenStorage;
    this.refreshRequest = {
      path: options.refreshRequest.path,
      method: options.refreshRequest.method ?? "GET",
      useRefreshToken: options.refreshRequest.useRefreshToken ?? true,
      body: options.refreshRequest.body,
      mapResponse: options.refreshRequest.mapResponse,
    };
    this.invalidTokenMatcher = options.invalidTokenMatcher ?? defaultInvalidTokenMatcher;

    this.client = axios.create({
      baseURL: options.baseURL,
      timeout: options.timeoutMs ?? 30_000,
    });
    this.client.interceptors.request.use(this.attachToken);
    this.client.interceptors.response.use((response) => response, this.handleError);
  }

  raw(): AxiosInstance {
    return this.client;
  }

  async get<T>(endpoint: string, params?: object, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.get(endpoint, { ...config, params });
    return response.data as T;
  }

  async post<T>(endpoint: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const headers =
      data instanceof FormData ? { ...(config?.headers ?? {}), "Content-Type": "multipart/form-data" } : config?.headers;
    const response = await this.client.post(endpoint, data, { ...config, headers });
    return response.data as T;
  }

  async put<T>(endpoint: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const headers =
      data instanceof FormData ? { ...(config?.headers ?? {}), "Content-Type": "multipart/form-data" } : config?.headers;
    const response = await this.client.put(endpoint, data, { ...config, headers });
    return response.data as T;
  }

  async delete<T>(endpoint: string, params?: object, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.delete(endpoint, { ...config, params });
    return response.data as T;
  }

  private processQueue(error: unknown, token: string | null = null): void {
    this.failedQueue.forEach((entry) => {
      if (token) {
        entry.resolve(token);
      } else {
        entry.reject(error);
      }
    });
    this.failedQueue = [];
  }

  private forceLogout(): void {
    this.tokenStorage.clear();
    dispatchAuthForceLogout();
  }

  private isRefreshRequest(config?: InternalAxiosRequestConfig): boolean {
    return !!config?.url && config.url === this.refreshRequest.path;
  }

  private attachToken = (config: InternalAxiosRequestConfig): InternalAxiosRequestConfig => {
    const useRefreshToken = this.isRefreshRequest(config) && this.refreshRequest.useRefreshToken;
    const token = useRefreshToken
      ? this.tokenStorage.getRefreshToken()
      : this.tokenStorage.getAccessToken();

    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    if (!config.headers["Content-Type"]) {
      config.headers["Content-Type"] = "application/json";
    }

    return config;
  };

  private handleError = async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig | undefined;

    if (this.invalidTokenMatcher(error) && error.response?.status === 403) {
      this.forceLogout();
      return Promise.reject(error);
    }

    if (originalRequest && error.response?.status === 401 && originalRequest._retry) {
      this.forceLogout();
      return Promise.reject(error);
    }

    if (
      !originalRequest ||
      error.response?.status !== 401 ||
      originalRequest._retry ||
      this.isRefreshRequest(originalRequest)
    ) {
      return Promise.reject(error);
    }

    originalRequest._retry = true;

    if (this.isRefreshing) {
      return new Promise<string>((resolve, reject) => {
        this.failedQueue.push({ resolve, reject });
      })
        .then((token) => {
          originalRequest.headers.Authorization = `Bearer ${token}`;
          return this.client(originalRequest);
        })
        .catch(() => Promise.reject(error));
    }

    this.isRefreshing = true;

    try {
      const tokenPair = await this.refreshTokens();
      this.processQueue(null, tokenPair.accessToken);
      originalRequest.headers.Authorization = `Bearer ${tokenPair.accessToken}`;
      return this.client(originalRequest);
    } catch (refreshError) {
      this.processQueue(refreshError, null);
      this.forceLogout();
      return Promise.reject(refreshError);
    } finally {
      this.isRefreshing = false;
    }
  };

  private async refreshTokens(): Promise<RefreshTokenPair> {
    const method = this.refreshRequest.method.toUpperCase();
    const response =
      method === "POST"
        ? await this.client.post(this.refreshRequest.path, this.refreshRequest.body ?? {})
        : await this.client.get(this.refreshRequest.path);

    const pair = this.refreshRequest.mapResponse(response.data);
    if (!pair.accessToken) {
      throw new Error("No access token in refresh response");
    }

    this.tokenStorage.setAccessToken(pair.accessToken);
    if (pair.refreshToken) {
      this.tokenStorage.setRefreshToken(pair.refreshToken);
    }
    return pair;
  }
}

export function createAuthenticatedAxiosClient(
  options: AuthenticatedAxiosClientOptions,
): AuthenticatedAxiosClient {
  return new AuthenticatedAxiosClient(options);
}
