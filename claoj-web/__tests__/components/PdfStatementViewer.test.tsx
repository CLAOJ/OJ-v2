import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import PdfStatementViewer from '@/components/ui/PdfStatementViewer';
import api from '@/lib/api';

// Isolate pdf.js worker + CSS side effects (import.meta.url / CSS can't be
// transformed by ts-jest).
jest.mock('@/components/ui/pdfSetup', () => ({}));

// Stub react-pdf: Document reports a 2-page load; Page renders a marker.
jest.mock('react-pdf', () => ({
    Document: ({ children, onLoadSuccess }: { children: React.ReactNode; onLoadSuccess?: (d: { numPages: number }) => void }) => {
        React.useEffect(() => { onLoadSuccess?.({ numPages: 2 }); }, [onLoadSuccess]);
        return <div data-testid="pdf-document">{children}</div>;
    },
    Page: ({ pageNumber }: { pageNumber: number }) => <div data-testid="pdf-page">page {pageNumber}</div>,
}));

// Mock the API client (default export) + the URL helper (named export).
jest.mock('@/lib/api', () => ({
    __esModule: true,
    default: { get: jest.fn() },
    problemPdfApi: { getPdfUrl: (code: string) => `http://test/api/problem/${code}/pdf` },
}));

const mockedGet = api.get as jest.Mock;

beforeAll(() => {
    // jsdom lacks object-URL APIs.
    (URL as unknown as { createObjectURL: unknown }).createObjectURL = jest.fn(() => 'blob:mock');
    (URL as unknown as { revokeObjectURL: unknown }).revokeObjectURL = jest.fn();
});

beforeEach(() => { mockedGet.mockReset(); });

test('shows a loading indicator while the PDF is being fetched', () => {
    mockedGet.mockReturnValue(new Promise(() => {})); // never resolves
    render(<PdfStatementViewer code="abc" />);
    expect(screen.getByText('pdfViewer.loading')).toBeInTheDocument();
});

test('renders the PDF pages after a successful fetch', async () => {
    mockedGet.mockResolvedValue({ data: new Blob(['%PDF'], { type: 'application/pdf' }) });
    render(<PdfStatementViewer code="abc" />);
    const pages = await screen.findAllByTestId('pdf-page');
    expect(pages).toHaveLength(2);
    expect(screen.getByText('pdfViewer.pageCount')).toBeInTheDocument();
});

test('shows a fallback with download + open links when the fetch fails', async () => {
    mockedGet.mockRejectedValue(new Error('403'));
    render(<PdfStatementViewer code="abc" />);
    await waitFor(() => expect(screen.getByText('pdfViewer.error')).toBeInTheDocument());
    const links = screen.getAllByRole('link');
    expect(links.length).toBeGreaterThanOrEqual(2);
    links.forEach((l) => expect(l).toHaveAttribute('href', 'http://test/api/problem/abc/pdf'));
});
