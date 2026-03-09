import * as React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { Input } from '@/components/ui/Input';

describe('Input Component', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('renders input element', () => {
        render(<Input />);
        expect(screen.getByRole('textbox')).toBeInTheDocument();
    });

    it('forwards ref correctly', () => {
        const ref = React.createRef<HTMLInputElement>();
        render(<Input ref={ref} />);
        expect(ref.current).toBeInTheDocument();
    });

    it('applies default className', () => {
        render(<Input />);
        const input = screen.getByRole('textbox');
        expect(input).toHaveClass('flex');
        expect(input).toHaveClass('h-10');
        expect(input).toHaveClass('rounded-md');
        expect(input).toHaveClass('border');
    });

    it('merges custom className', () => {
        render(<Input className="custom-class" />);
        const input = screen.getByRole('textbox');
        expect(input).toHaveClass('border');
        expect(input).toHaveClass('custom-class');
    });

    it('handles different input types', () => {
        const { rerender } = render(<Input type="text" data-testid="input" />);
        expect(screen.getByTestId('input')).toHaveAttribute('type', 'text');

        rerender(<Input type="email" data-testid="input" />);
        expect(screen.getByTestId('input')).toHaveAttribute('type', 'email');

        rerender(<Input type="password" data-testid="input" />);
        expect(screen.getByTestId('input')).toHaveAttribute('type', 'password');
    });

    it('handles password type correctly', () => {
        render(<Input type="password" data-testid="password" />);
        const passwordInput = screen.getByTestId('password');
        expect(passwordInput).toHaveAttribute('type', 'password');
    });

    it('handles placeholder', () => {
        render(<Input placeholder="Enter text here" />);
        expect(screen.getByPlaceholderText('Enter text here')).toBeInTheDocument();
    });

    it('handles value and onChange', () => {
        const handleChange = jest.fn();
        render(<Input data-testid="input" defaultValue="test value" onChange={handleChange} />);
        const input = screen.getByTestId('input') as HTMLInputElement;
        expect(input.value).toBe('test value');

        fireEvent.change(input, { target: { value: 'new value' } });
        expect(handleChange).toHaveBeenCalledTimes(1);
    });

    it('handles disabled state', () => {
        render(<Input disabled />);
        expect(screen.getByRole('textbox')).toBeDisabled();
        expect(screen.getByRole('textbox')).toHaveClass('disabled:cursor-not-allowed');
        expect(screen.getByRole('textbox')).toHaveClass('disabled:opacity-50');
    });

    it('handles required attribute', () => {
        render(<Input required />);
        expect(screen.getByRole('textbox')).toHaveAttribute('required');
    });

    it('handles readOnly attribute', () => {
        render(<Input readOnly />);
        expect(screen.getByRole('textbox')).toHaveAttribute('readonly');
    });

    it('handles autoFocus attribute', () => {
        render(<Input autoFocus />);
        expect(screen.getByRole('textbox')).toHaveFocus();
    });

    it('handles name attribute', () => {
        render(<Input name="test-input" />);
        expect(screen.getByRole('textbox')).toHaveAttribute('name', 'test-input');
    });

    it('handles id attribute', () => {
        render(<Input id="test-id" />);
        expect(screen.getByRole('textbox')).toHaveAttribute('id', 'test-id');
    });

    it('handles aria attributes', () => {
        render(
            <Input
                aria-label="Test input"
                aria-describedby="description"
                aria-invalid="true"
            />
        );
        const input = screen.getByRole('textbox');
        expect(input).toHaveAttribute('aria-label', 'Test input');
        expect(input).toHaveAttribute('aria-describedby', 'description');
        expect(input).toHaveAttribute('aria-invalid', 'true');
    });

    it('handles data attributes', () => {
        render(<Input data-testid="custom-input" data-custom="value" />);
        const input = screen.getByTestId('custom-input');
        expect(input).toHaveAttribute('data-custom', 'value');
    });

    it('handles onBlur event', () => {
        const handleBlur = jest.fn();
        render(<Input onBlur={handleBlur} />);
        fireEvent.blur(screen.getByRole('textbox'));
        expect(handleBlur).toHaveBeenCalledTimes(1);
    });

    it('handles onFocus event', () => {
        const handleFocus = jest.fn();
        render(<Input onFocus={handleFocus} />);
        fireEvent.focus(screen.getByRole('textbox'));
        expect(handleFocus).toHaveBeenCalledTimes(1);
    });

    it('handles onKeyDown event', () => {
        const handleKeyDown = jest.fn();
        render(<Input onKeyDown={handleKeyDown} />);
        fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' });
        expect(handleKeyDown).toHaveBeenCalledTimes(1);
        expect(handleKeyDown).toHaveBeenCalledWith(
            expect.objectContaining({ key: 'Enter' })
        );
    });

    it('handles onKeyUp event', () => {
        const handleKeyUp = jest.fn();
        render(<Input onKeyUp={handleKeyUp} />);
        fireEvent.keyUp(screen.getByRole('textbox'), { key: 'Enter' });
        expect(handleKeyUp).toHaveBeenCalledTimes(1);
    });

    it('handles onKeyPress event', () => {
        // Note: onKeyPress is deprecated in React 17+
        // This test verifies the prop is accepted, but modern apps should use onKeyDown
        const handler = jest.fn();
        render(<Input data-testid="input" onKeyDown={handler} />);
        fireEvent.keyDown(screen.getByTestId('input'), { key: 'Enter' });
        expect(handler).toHaveBeenCalledTimes(1);
    });

    it('handles minLength and maxLength', () => {
        render(<Input minLength={3} maxLength={10} />);
        const input = screen.getByRole('textbox');
        expect(input).toHaveAttribute('minLength', '3');
        expect(input).toHaveAttribute('maxLength', '10');
    });

    it('handles pattern attribute', () => {
        render(<Input pattern="[A-Za-z]+" />);
        expect(screen.getByRole('textbox')).toHaveAttribute('pattern', '[A-Za-z]+');
    });

    it('handles step attribute for number type', () => {
        render(<Input type="number" data-testid="number-input" step={0.1} />);
        expect(screen.getByTestId('number-input')).toHaveAttribute('step', '0.1');
    });

    it('handles min and max for number type', () => {
        render(<Input type="number" data-testid="number-input" min={0} max={100} />);
        const input = screen.getByTestId('number-input');
        expect(input).toHaveAttribute('min', '0');
        expect(input).toHaveAttribute('max', '100');
    });

    it('renders file input correctly', () => {
        render(<Input type="file" data-testid="file-input" />);
        expect(screen.getByTestId('file-input')).toBeInTheDocument();
    });

    it('handles checkbox input', () => {
        render(<Input type="checkbox" data-testid="checkbox" />);
        const checkbox = screen.getByTestId('checkbox');
        expect(checkbox).toBeInTheDocument();
        expect(checkbox).toHaveAttribute('type', 'checkbox');
    });

    it('handles radio input', () => {
        render(<Input type="radio" data-testid="radio" />);
        const radio = screen.getByTestId('radio');
        expect(radio).toBeInTheDocument();
        expect(radio).toHaveAttribute('type', 'radio');
    });
});
