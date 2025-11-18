import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useLangFlowHealth, useLangFlowFlows, useLangFlowFlow } from '../use-langflow';
import { langflowApi } from '../../api/langflow';
import type { LangFlowFlow } from '@/types/api/sessions';

// Mock the langflowApi module
jest.mock('../../api/langflow', () => ({
  langflowApi: {
    checkHealth: jest.fn(),
    listFlows: jest.fn(),
    getFlow: jest.fn(),
  },
}));

const mockLangflowApi = langflowApi as jest.Mocked<typeof langflowApi>;

// Helper to create a wrapper with QueryClient
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        staleTime: 0,
        gcTime: 0,
      },
    },
  });

  function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  }
  Wrapper.displayName = 'TestQueryClientWrapper';

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return Wrapper as any;
};

describe('useLangFlowHealth', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return health status when service is healthy', async () => {
    mockLangflowApi.checkHealth.mockResolvedValue(true);

    const { result } = renderHook(() => useLangFlowHealth(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toBe(true);
    expect(mockLangflowApi.checkHealth).toHaveBeenCalled();
  });

  it('should return false when service is unhealthy', async () => {
    mockLangflowApi.checkHealth.mockResolvedValue(false);

    const { result } = renderHook(() => useLangFlowHealth(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toBe(false);
  });

  it('should handle errors gracefully', async () => {
    mockLangflowApi.checkHealth.mockRejectedValue(new Error('Network error'));

    const { result } = renderHook(() => useLangFlowHealth(), {
      wrapper: createWrapper(),
    });

    // Wait for loading to finish and error state to be set
    await waitFor(() => expect(result.current.isLoading).toBe(false), { timeout: 3000 });
    await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 3000 });

    expect(result.current.error).toBeDefined();
  });
});

describe('useLangFlowFlows', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return list of flows', async () => {
    const mockFlows: LangFlowFlow[] = [
      {
        id: 'flow-1',
        name: 'Test Flow 1',
        description: 'Description 1',
        data: {},
        updated_at: '2025-01-01T00:00:00Z',
      },
      {
        id: 'flow-2',
        name: 'Test Flow 2',
        description: 'Description 2',
        data: {},
        updated_at: '2025-01-02T00:00:00Z',
      },
    ];

    mockLangflowApi.listFlows.mockResolvedValue(mockFlows);

    const { result } = renderHook(() => useLangFlowFlows(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockFlows);
    expect(mockLangflowApi.listFlows).toHaveBeenCalled();
  });

  it('should return empty array when no flows exist', async () => {
    mockLangflowApi.listFlows.mockResolvedValue([]);

    const { result } = renderHook(() => useLangFlowFlows(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual([]);
  });

  it('should handle errors', async () => {
    mockLangflowApi.listFlows.mockRejectedValue(new Error('Failed to fetch flows'));

    const { result } = renderHook(() => useLangFlowFlows(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeDefined();
  });
});

describe('useLangFlowFlow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return a single flow', async () => {
    const mockFlow: LangFlowFlow = {
      id: 'flow-1',
      name: 'Test Flow',
      description: 'Test description',
      data: { nodes: [], edges: [] },
      updated_at: '2025-01-01T00:00:00Z',
    };

    mockLangflowApi.getFlow.mockResolvedValue(mockFlow);

    const { result } = renderHook(() => useLangFlowFlow('flow-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockFlow);
    expect(mockLangflowApi.getFlow).toHaveBeenCalledWith('flow-1');
  });

  it('should not fetch when id is empty', () => {
    const { result } = renderHook(() => useLangFlowFlow(''), {
      wrapper: createWrapper(),
    });

    expect(result.current.isFetching).toBe(false);
    expect(mockLangflowApi.getFlow).not.toHaveBeenCalled();
  });

  it('should handle errors', async () => {
    mockLangflowApi.getFlow.mockRejectedValue(new Error('Flow not found'));

    const { result } = renderHook(() => useLangFlowFlow('non-existent'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeDefined();
  });
});
