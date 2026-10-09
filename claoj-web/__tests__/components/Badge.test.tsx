import * as React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from '@jest/globals';
import { Badge } from '@/components/ui/Badge';

describe('Badge', () => {
    it('is a square-ish tag, not a pill', () => {
        render(<Badge>AC</Badge>);
        const badge = screen.getByText('AC');
        expect(badge).toHaveClass('rounded');
        expect(badge).not.toHaveClass('rounded-full');
        expect(badge).toHaveClass('px-1.5');
    });
});
