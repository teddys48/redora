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

export interface KeyItem {
  key: string;
  type: string;
  ttl: number;
  size?: number;
}

export interface ScanKeysResponse {
  keys: KeyItem[];
  nextCursor: number;
}

export interface KeyDetailResponse {
  key: string;
  type: string;
  ttl: number;
  value: any;
}

export interface CreateKeyInput {
  key: string;
  type: string;
  value: any;
  ttl?: number;
  field?: string;
  score?: number;
}

export interface CommandResponse {
  result: string;
  execution: string;
  status: string;
}

export interface PubSubChannelsResponse {
  channels: string[];
}

export interface PubSubPublishResponse {
  message: string;
  subscribers: number;
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
        // Fallback
      }
      throw new Error(errorMessage);
    }

    return response.json() as Promise<T>;
  }

  public async getHealth(): Promise<HealthStatus> {
    return this.request<HealthStatus>('/api/health');
  }

  // Connection Management
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

  // Key Browser & CRUD
  public async scanKeys(connId: string, pattern = '*', type = '', cursor = 0, limit = 100): Promise<ScanKeysResponse> {
    const query = new URLSearchParams({
      pattern,
      type,
      cursor: cursor.toString(),
      limit: limit.toString(),
    });
    return this.request<ScanKeysResponse>(`/api/connections/${connId}/keys?${query.toString()}`);
  }

  public async getKeyDetail(connId: string, key: string): Promise<KeyDetailResponse> {
    const encodedKey = encodeURIComponent(key);
    const res = await this.request<{ data: KeyDetailResponse }>(`/api/connections/${connId}/keys/${encodedKey}`);
    return res.data;
  }

  public async createKey(connId: string, payload: CreateKeyInput): Promise<void> {
    await this.request(`/api/connections/${connId}/keys`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  public async updateKey(connId: string, key: string, payload: CreateKeyInput): Promise<void> {
    const encodedKey = encodeURIComponent(key);
    await this.request(`/api/connections/${connId}/keys/${encodedKey}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
  }

  public async deleteKey(connId: string, key: string): Promise<void> {
    const encodedKey = encodeURIComponent(key);
    await this.request(`/api/connections/${connId}/keys/${encodedKey}`, {
      method: 'DELETE',
    });
  }

  public async renameKey(connId: string, key: string, newKey: string): Promise<void> {
    const encodedKey = encodeURIComponent(key);
    await this.request(`/api/connections/${connId}/keys/${encodedKey}/rename`, {
      method: 'POST',
      body: JSON.stringify({ newKey }),
    });
  }

  public async setTTL(connId: string, key: string, ttl: number): Promise<void> {
    const encodedKey = encodeURIComponent(key);
    await this.request(`/api/connections/${connId}/keys/${encodedKey}/expire`, {
      method: 'POST',
      body: JSON.stringify({ ttl }),
    });
  }

  // CLI Console
  public async executeCommand(connId: string, command: string): Promise<CommandResponse> {
    return this.request<CommandResponse>(`/api/connections/${connId}/command`, {
      method: 'POST',
      body: JSON.stringify({ command }),
    });
  }

  // Pub/Sub
  public async getPubSubChannels(connId: string, pattern = '*'): Promise<PubSubChannelsResponse> {
    const query = new URLSearchParams({ pattern });
    return this.request<PubSubChannelsResponse>(`/api/connections/${connId}/pubsub/channels?${query.toString()}`);
  }

  public async publishMessage(connId: string, channel: string, message: string): Promise<PubSubPublishResponse> {
    return this.request<PubSubPublishResponse>(`/api/connections/${connId}/pubsub/publish`, {
      method: 'POST',
      body: JSON.stringify({ channel, message }),
    });
  }
}

export const apiClient = new APIClient();
