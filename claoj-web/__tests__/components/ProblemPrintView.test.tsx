import React from 'react';
import { render, screen } from '@testing-library/react';
import ProblemPrintView from '@/app/[locale]/problem/[code]/print/ProblemPrintView';
import type { ProblemDetail } from '@/types';

jest.mock('@/components/ui/MathRenderer', () => ({
    __esModule: true,
    default: ({ content }: { content: string }) => <div data-testid="statement">{content}</div>,
}));

const problem: ProblemDetail = {
    id: 1,
    code: 'aplusb',
    name: 'A+B',
    description: 'Add **two** numbers.',
    is_full_markup: false,
    points: 100,
    partial: false,
    user_count: 1,
    ac_rate: 1,
    group: 'Basics',
    is_solved: false,
    is_attempted: false,
    time_limit: 1,
    memory_limit: 256,
    languages: [],
    types: [],
    authors: [{ username: 'alice' }],
    pdf_url: '/problem/aplusb/pdf',
};

beforeEach(() => {
    jest.spyOn(window, 'print').mockImplementation(() => {});
});

afterEach(() => {
    jest.restoreAllMocks();
});

test('raw mode renders the bare statement without chrome and never auto-prints', () => {
    render(<ProblemPrintView problem={problem} autoPrint raw />);

    expect(screen.getByTestId('statement')).toHaveTextContent('Add **two** numbers.');
    expect(screen.getByText('A+B')).toBeInTheDocument();
    // Back/print controls and the uploaded-PDF note belong to the print view.
    expect(screen.queryByText('print')).not.toBeInTheDocument();
    expect(screen.queryByText('uploadedPdfNote')).not.toBeInTheDocument();
    expect(screen.queryByText(/@alice/)).not.toBeInTheDocument();
    expect(window.print).not.toHaveBeenCalled();
});

test('the print view keeps its controls and still auto-prints when asked', () => {
    render(<ProblemPrintView problem={problem} autoPrint />);

    // The navigation Link is globally mocked to render nothing, so the print
    // button is the reliable control marker here.
    expect(screen.getByText('print')).toBeInTheDocument();
    expect(screen.getByText('uploadedPdfNote')).toBeInTheDocument();
    expect(window.print).toHaveBeenCalled();
});

