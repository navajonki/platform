import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { LangFlowSelector } from '../langflow-selector';
import { useLangFlowFlows, useLangFlowFlow } from '@/services/queries/use-langflow';
import type { LangFlowFlow } from '@/types/api/sessions';

// Mock the hooks
jest.mock('@/services/queries/use-langflow', () => ({
  useLangFlowFlows: jest.fn(),
  useLangFlowFlow: jest.fn(),
}));

const mockUseLangFlowFlows = useLangFlowFlows as jest.MockedFunction<typeof useLangFlowFlows>;
const mockUseLangFlowFlow = useLangFlowFlow as jest.MockedFunction<typeof useLangFlowFlow>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
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

const mockFlows: LangFlowFlow[] = [
  {
    id: 'flow-1',
    name: 'Test Flow 1',
    description: 'Description for flow 1',
    data: {},
    updated_at: '2025-01-01T12:00:00Z',
  },
  {
    id: 'flow-2',
    name: 'Test Flow 2',
    description: 'Description for flow 2',
    data: {},
    updated_at: '2025-01-02T12:00:00Z',
  },
];

describe('LangFlowSelector', () => {
  const mockOnChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should show loading skeleton while fetching flows', () => {
    mockUseLangFlowFlows.mockReturnValue({
      data: undefined,
      isLoading: true,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
    mockUseLangFlowFlow.mockReturnValue({
      data: undefined,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    const { container } = render(
      <LangFlowSelector flowId="" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    // Check for skeleton elements (they have specific classes from Shadcn)
    const skeletons = container.querySelectorAll('[class*="animate-pulse"]');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('should render flow dropdown with available flows', async () => {
    mockUseLangFlowFlows.mockReturnValue({
      data: mockFlows,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
    mockUseLangFlowFlow.mockReturnValue({
      data: undefined,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <LangFlowSelector flowId="" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    expect(screen.getByText('Select LangFlow Workflow')).toBeInTheDocument();
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  it('should render flow dropdown', () => {
    mockUseLangFlowFlows.mockReturnValue({
      data: mockFlows,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
    mockUseLangFlowFlow.mockReturnValue({
      data: undefined,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <LangFlowSelector flowId="" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    // Check that the select component is rendered
    const select = screen.getByRole('combobox');
    expect(select).toBeInTheDocument();
    expect(screen.getByText('Choose a workflow...')).toBeInTheDocument();
  });

  it('should display selected flow details', () => {
    mockUseLangFlowFlows.mockReturnValue({
      data: mockFlows,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
    mockUseLangFlowFlow.mockReturnValue({
      data: mockFlows[0],
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <LangFlowSelector flowId="flow-1" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    // Check for flow details in the card (not the dropdown)
    expect(screen.getByText('Description for flow 1')).toBeInTheDocument();
    expect(screen.getByText(/Last updated:/)).toBeInTheDocument();
  });

  it('should show link to edit flow in LangFlow', () => {
    mockUseLangFlowFlows.mockReturnValue({
      data: mockFlows,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
    mockUseLangFlowFlow.mockReturnValue({
      data: mockFlows[0],
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <LangFlowSelector flowId="flow-1" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    const editLink = screen.getByText('Edit in LangFlow').closest('a');
    expect(editLink).toHaveAttribute('href', 'http://localhost:7860/flow/flow-1');
    expect(editLink).toHaveAttribute('target', '_blank');
    expect(editLink).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('should show empty state when no flows are available', () => {
    mockUseLangFlowFlows.mockReturnValue({
      data: [],
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
    mockUseLangFlowFlow.mockReturnValue({
      data: undefined,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <LangFlowSelector flowId="" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    expect(screen.getByText(/No workflows found/i)).toBeInTheDocument();
    expect(screen.getByText('LangFlow')).toBeInTheDocument();
  });

  it('should show "No description available" when flow has no description', () => {
    const flowWithoutDescription: LangFlowFlow = {
      ...mockFlows[0],
      description: '',
    };

    mockUseLangFlowFlows.mockReturnValue({
      data: mockFlows,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
    mockUseLangFlowFlow.mockReturnValue({
      data: flowWithoutDescription,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <LangFlowSelector flowId="flow-1" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    expect(screen.getByText('No description available')).toBeInTheDocument();
  });
});
