import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LangFlowInputConfig } from '../langflow-input-config';
import type { LangFlowInput } from '@/types/api/sessions';

describe('LangFlowInputConfig', () => {
  const mockOnChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render the input configuration card', () => {
    const mockInput: LangFlowInput = { message: '' };

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    expect(screen.getByText('Flow Input Configuration')).toBeInTheDocument();
    expect(screen.getByText(/Configure inputs for the LangFlow workflow/i)).toBeInTheDocument();
  });

  it('should render message textarea with label', () => {
    const mockInput: LangFlowInput = { message: '' };

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    expect(screen.getByLabelText('Message')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Enter your message for the workflow/i)).toBeInTheDocument();
  });

  it('should display current message value', () => {
    const mockInput: LangFlowInput = { message: 'Test message content' };

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    const textarea = screen.getByRole('textbox');
    expect(textarea).toHaveValue('Test message content');
  });

  it('should call onChange when message is typed', async () => {
    const user = userEvent.setup();
    const mockInput: LangFlowInput = { message: '' };

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    const textarea = screen.getByRole('textbox');
    await user.type(textarea, 'N');

    // onChange is called for the character typed
    expect(mockOnChange).toHaveBeenCalled();
    expect(mockOnChange).toHaveBeenCalledWith(expect.objectContaining({ message: expect.stringContaining('N') }));
  });

  it('should call onChange with updated message when text is changed', async () => {
    const user = userEvent.setup();
    const mockInput: LangFlowInput = { message: 'Initial message' };

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    const textarea = screen.getByRole('textbox');
    await user.clear(textarea);

    // After clear, onChange should have been called
    expect(mockOnChange).toHaveBeenCalled();

    // After typing, onChange should include the new characters
    await user.type(textarea, 'U');
    expect(mockOnChange).toHaveBeenCalledWith(expect.objectContaining({ message: expect.stringContaining('U') }));
  });

  it('should preserve other properties in input object when updating message', async () => {
    const user = userEvent.setup();
    const mockInput: LangFlowInput = {
      message: 'Test',
      someOtherField: 'value',
    };

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    const textarea = screen.getByRole('textbox');
    await user.type(textarea, 'X');

    // Should spread existing properties and update message
    expect(mockOnChange).toHaveBeenCalledWith(
      expect.objectContaining({
        someOtherField: 'value',
        message: expect.any(String),
      })
    );
  });

  it('should handle empty message value', () => {
    const mockInput: LangFlowInput = { message: '' };

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    const textarea = screen.getByRole('textbox');
    expect(textarea).toHaveValue('');
  });

  it('should handle undefined message value', () => {
    const mockInput: LangFlowInput = {};

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    const textarea = screen.getByRole('textbox');
    expect(textarea).toHaveValue('');
  });

  it('should show helper text', () => {
    const mockInput: LangFlowInput = { message: '' };

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    expect(screen.getByText(/This message will be passed to the LangFlow workflow as input/i)).toBeInTheDocument();
  });

  it('should render textarea with correct number of rows', () => {
    const mockInput: LangFlowInput = { message: '' };

    render(
      <LangFlowInputConfig value={mockInput} onChange={mockOnChange} />
    );

    const textarea = screen.getByRole('textbox');
    expect(textarea).toHaveAttribute('rows', '4');
  });
});
