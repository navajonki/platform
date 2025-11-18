/**
 * LangFlow API service
 * Provides methods for interacting with LangFlow flows
 */

import { apiClient } from './client';
import type { LangFlowFlow } from '@/types/api/sessions';

type ListFlowsResponse = {
  flows: LangFlowFlow[];
};

type GetFlowResponse = LangFlowFlow;

type HealthResponse = {
  available: boolean;
  status: string;
};

/**
 * LangFlow API methods
 */
export const langflowApi = {
  /**
   * List all available LangFlow flows
   */
  async listFlows(): Promise<LangFlowFlow[]> {
    const response = await apiClient.get<ListFlowsResponse>('/langflow/flows');
    return response.flows || [];
  },

  /**
   * Get a specific flow by ID
   */
  async getFlow(id: string): Promise<LangFlowFlow> {
    return apiClient.get<GetFlowResponse>(`/langflow/flows/${id}`);
  },

  /**
   * Check LangFlow service health
   */
  async checkHealth(): Promise<boolean> {
    try {
      const response = await apiClient.get<HealthResponse>('/langflow/health');
      return response.available === true && response.status === 'healthy';
    } catch {
      return false;
    }
  },
};
