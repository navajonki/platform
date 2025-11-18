import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SessionTypeSelector } from '../session-type-selector';
import { useLangFlowHealth } from '@/services/queries/use-langflow';

// Mock the useLangFlowHealth hook
jest.mock('@/services/queries/use-langflow', () => ({
  useLangFlowHealth: jest.fn(),
}));

const mockUseLangFlowHealth = useLangFlowHealth as jest.MockedFunction<typeof useLangFlowHealth>;

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

describe('SessionTypeSelector', () => {
  const mockOnChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render both session type options', () => {
    mockUseLangFlowHealth.mockReturnValue({
      data: true,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <SessionTypeSelector value="claude-code" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    expect(screen.getByText('Claude Code Agent (Markdown)')).toBeInTheDocument();
    expect(screen.getByText('LangFlow Visual Workflow')).toBeInTheDocument();
  });

  it('should show Claude Code as selected by default', () => {
    mockUseLangFlowHealth.mockReturnValue({
      data: true,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <SessionTypeSelector value="claude-code" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    const claudeCodeRadio = screen.getByRole('radio', { name: /claude code/i });
    expect(claudeCodeRadio).toBeChecked();
  });

  it('should call onChange when LangFlow option is selected', async () => {
    const user = userEvent.setup();
    mockUseLangFlowHealth.mockReturnValue({
      data: true,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <SessionTypeSelector value="claude-code" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    const langflowRadio = screen.getByRole('radio', { name: /langflow/i });
    await user.click(langflowRadio);

    expect(mockOnChange).toHaveBeenCalledWith('langflow');
  });

  it('should disable LangFlow option when service is unhealthy', () => {
    mockUseLangFlowHealth.mockReturnValue({
      data: false,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <SessionTypeSelector value="claude-code" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    const langflowRadio = screen.getByRole('radio', { name: /langflow/i });
    expect(langflowRadio).toBeDisabled();
  });

  it('should show warning when LangFlow is unavailable', async () => {
    mockUseLangFlowHealth.mockReturnValue({
      data: false,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    render(
      <SessionTypeSelector value="claude-code" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(screen.getByText(/LangFlow service is not available/i)).toBeInTheDocument();
    });
  });

  it('should show health check icon when LangFlow is healthy', () => {
    mockUseLangFlowHealth.mockReturnValue({
      data: true,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    const { container } = render(
      <SessionTypeSelector value="claude-code" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    // Look for the CheckCircle2 icon (green check mark)
    const checkIcon = container.querySelector('.text-green-500');
    expect(checkIcon).toBeInTheDocument();
  });

  it('should show warning icon when LangFlow is unhealthy', () => {
    mockUseLangFlowHealth.mockReturnValue({
      data: false,
      isLoading: false,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    const { container } = render(
      <SessionTypeSelector value="claude-code" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    // Look for the AlertCircle icon (yellow warning)
    const warningIcon = container.querySelector('.text-yellow-500');
    expect(warningIcon).toBeInTheDocument();
  });

  it('should not show health icons while loading', () => {
    mockUseLangFlowHealth.mockReturnValue({
      data: undefined,
      isLoading: true,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);

    const { container } = render(
      <SessionTypeSelector value="claude-code" onChange={mockOnChange} />,
      { wrapper: createWrapper() }
    );

    const checkIcon = container.querySelector('.text-green-500');
    const warningIcon = container.querySelector('.text-yellow-500');

    expect(checkIcon).not.toBeInTheDocument();
    expect(warningIcon).not.toBeInTheDocument();
  });
});
