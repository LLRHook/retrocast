interface ApiError {
  code: string;
  message: string;
}

interface ApiErrorResponse {
  error: ApiError;
}

export class ApiClientError extends Error {
  code: string;
  status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.name = "ApiClientError";
    this.code = code;
    this.status = status;
  }
}

type TokenProvider = () => string | null;
type TokenRefresher = () => Promise<string>;

class ApiClient {
  private tokenProvider: TokenProvider = () => null;
  private tokenRefresher: TokenRefresher | null = null;
  private refreshPromise: Promise<string> | null = null;

  get baseUrl(): string {
    return localStorage.getItem("serverUrl") || "";
  }

  setTokenProvider(provider: TokenProvider) {
    this.tokenProvider = provider;
  }

  setTokenRefresher(refresher: TokenRefresher) {
    this.tokenRefresher = refresher;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    retry = true,
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    const token = this.tokenProvider();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const res = await fetch(url, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });

    if (res.status === 401 && retry && this.tokenRefresher) {
      const newToken = await this.refreshToken();
      if (newToken) {
        headers["Authorization"] = `Bearer ${newToken}`;
        const retryRes = await fetch(url, {
          method,
          headers,
          body: body !== undefined ? JSON.stringify(body) : undefined,
        });
        return this.handleResponse<T>(retryRes);
      }
    }

    return this.handleResponse<T>(res);
  }

  private async refreshToken(): Promise<string | null> {
    if (!this.tokenRefresher) return null;

    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    this.refreshPromise = this.tokenRefresher();
    try {
      return await this.refreshPromise;
    } catch {
      return null;
    } finally {
      this.refreshPromise = null;
    }
  }

  private async handleResponse<T>(res: Response): Promise<T> {
    if (!res.ok) {
      let errorData: ApiErrorResponse | undefined;
      try {
        errorData = await res.json();
      } catch {
        // Response body is not JSON
      }
      throw new ApiClientError(
        errorData?.error?.code || "UNKNOWN_ERROR",
        errorData?.error?.message || `Request failed with status ${res.status}`,
        res.status,
      );
    }

    const json = await res.json();
    return json.data as T;
  }

  get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  put<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("PUT", path, body);
  }

  patch<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("PATCH", path, body);
  }

  delete<T>(path: string): Promise<T> {
    return this.request<T>("DELETE", path);
  }

  async upload<T>(path: string, file: File): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = {};
    const token = this.tokenProvider();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch(url, {
      method: "POST",
      headers,
      body: formData,
    });
    if (!res.ok) {
      let errorData: ApiErrorResponse | undefined;
      try {
        errorData = await res.json();
      } catch {
        // not JSON
      }
      throw new ApiClientError(
        errorData?.error?.code || "UNKNOWN_ERROR",
        errorData?.error?.message || `Upload failed with status ${res.status}`,
        res.status,
      );
    }
    return (await res.json()) as T;
  }
}

export const api = new ApiClient();
