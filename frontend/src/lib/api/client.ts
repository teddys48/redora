export interface HealthStatus {
  status: string;
  timestamp: string;
  version: string;
}

export interface RedisConnection {
  id: string;
  name: string;
  host: string;
  port: number;
  username?: string;
  db: number;
  tlsEnabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ConnectionCreateInput {
  name: string;
  host: string;
  port: number;
  username?: string;
  password?: string;
  db: number;
  tlsEnabled: boolean;
}

export interface APIErrorResponse {
  error: {
    code: string;
    message: string;
  };
}

class APIClient {
  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(endpoint, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
      try {
        const errorData = (await response.json()) as APIErrorResponse;
        if (errorData?.error?.message) {
          errorMessage = errorData.error.message;
        }
      } catch {
        // Fallback to default HTTP status text
      }
      throw new Error(errorMessage);
    }

    return response.json() as Promise<T>;
  }

  public async getHealth(): Promise<HealthStatus> {
    return this.request<HealthStatus>('/api/health');
  }

  public async getConnections(): Promise<RedisConnection[]> {
    const res = await this.request<{ data: RedisConnection[] }>('/api/connections');
    return res.data;
  }

  public async createConnection(data: ConnectionCreateInput): Promise<RedisConnection> {
    const res = await this.request<{ data: RedisConnection }>('/api/connections', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return res.data;
  }

  public async updateConnection(id: string, data: Partial<ConnectionCreateInput>): Promise<RedisConnection> {
    const res = await this.request<{ data: RedisConnection }>(`/api/connections/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
    return res.data;
  }

  public async deleteConnection(id: string): Promise<void> {
    await this.request<{ message: string }>(`/api/connections/${id}`, {
      method: 'DELETE',
    });
  }

  public async testConnection(id: string): Promise<{ status: string; message: string }> {
    return this.request<{ status: string; message: string }>(`/api/connections/${id}/test`, {
      method: 'POST',
    });
  }

  public async testConnectionInput(data: ConnectionCreateInput): Promise<{ status: string; message: string }> {
    return this.request<{ status: string; message: string }>('/api/connections/test', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }
}

export const apiClient = new APIClient();
