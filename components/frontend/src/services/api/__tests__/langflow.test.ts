import { langflowApi } from '../langflow';
import { apiClient } from '../client';
import type { LangFlowFlow } from '@/types/api/sessions';

// Mock the apiClient module
jest.mock('../client', () => ({
  apiClient: {
    get: jest.fn(),
  },
}));

describe('langflowApi', () => {
  const mockApiClient = apiClient as jest.Mocked<typeof apiClient>;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listFlows', () => {
    it('should return flows from the API response', async () => {
      const mockFlows: LangFlowFlow[] = [
        {
          id: 'flow-1',
          name: 'Test Flow 1',
          description: 'Test description 1',
          data: {},
          updated_at: '2025-01-01T00:00:00Z',
        },
        {
          id: 'flow-2',
          name: 'Test Flow 2',
          description: 'Test description 2',
          data: {},
          updated_at: '2025-01-02T00:00:00Z',
        },
      ];

      mockApiClient.get.mockResolvedValue({ flows: mockFlows });

      const result = await langflowApi.listFlows();

      expect(mockApiClient.get).toHaveBeenCalledWith('/langflow/flows');
      expect(result).toEqual(mockFlows);
    });

    it('should return empty array when flows is undefined', async () => {
      mockApiClient.get.mockResolvedValue({});

      const result = await langflowApi.listFlows();

      expect(result).toEqual([]);
    });

    it('should throw error when API call fails', async () => {
      const error = new Error('Network error');
      mockApiClient.get.mockRejectedValue(error);

      await expect(langflowApi.listFlows()).rejects.toThrow('Network error');
    });
  });

  describe('getFlow', () => {
    it('should return a single flow from the API', async () => {
      const mockFlow: LangFlowFlow = {
        id: 'flow-1',
        name: 'Test Flow',
        description: 'Test description',
        data: { nodes: [], edges: [] },
        updated_at: '2025-01-01T00:00:00Z',
      };

      mockApiClient.get.mockResolvedValue(mockFlow);

      const result = await langflowApi.getFlow('flow-1');

      expect(mockApiClient.get).toHaveBeenCalledWith('/langflow/flows/flow-1');
      expect(result).toEqual(mockFlow);
    });

    it('should throw error when flow not found', async () => {
      const error = new Error('Flow not found');
      mockApiClient.get.mockRejectedValue(error);

      await expect(langflowApi.getFlow('non-existent')).rejects.toThrow('Flow not found');
    });
  });

  describe('checkHealth', () => {
    it('should return true when LangFlow is healthy', async () => {
      mockApiClient.get.mockResolvedValue({ status: 'ok' });

      const result = await langflowApi.checkHealth();

      expect(mockApiClient.get).toHaveBeenCalledWith('/langflow/health');
      expect(result).toBe(true);
    });

    it('should return false when health check fails', async () => {
      mockApiClient.get.mockRejectedValue(new Error('Service unavailable'));

      const result = await langflowApi.checkHealth();

      expect(result).toBe(false);
    });

    it('should return false when status is not ok', async () => {
      mockApiClient.get.mockResolvedValue({ status: 'error' });

      const result = await langflowApi.checkHealth();

      expect(result).toBe(false);
    });
  });
});
