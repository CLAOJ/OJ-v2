import * as React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from '@jest/globals';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/Card';

describe('Card Components', () => {
    it('renders Card with proper structure', () => {
        render(
            <Card>
                <CardHeader>
                    <CardTitle>Title</CardTitle>
                    <CardDescription>Description</CardDescription>
                </CardHeader>
                <CardContent>Content</CardContent>
                <CardFooter>Footer</CardFooter>
            </Card>
        );

        expect(screen.getByText('Title')).toBeInTheDocument();
        expect(screen.getByText('Description')).toBeInTheDocument();
        expect(screen.getByText('Content')).toBeInTheDocument();
        expect(screen.getByText('Footer')).toBeInTheDocument();
    });

    it('renders CardHeader correctly', () => {
        render(<CardHeader data-testid="header">Header Content</CardHeader>);
        const header = screen.getByTestId('header');
        expect(header).toBeInTheDocument();
        expect(header).toHaveClass('flex');
        expect(header).toHaveClass('flex-col');
    });

    it('renders CardTitle correctly', () => {
        render(<CardTitle data-testid="title">Test Title</CardTitle>);
        const title = screen.getByTestId('title');
        expect(title).toBeInTheDocument();
        expect(title.tagName).toBe('H3');
        expect(title).toHaveClass('font-semibold');
        expect(title).toHaveClass('leading-none');
    });

    it('renders CardDescription correctly', () => {
        render(<CardDescription data-testid="description">Test Description</CardDescription>);
        const desc = screen.getByTestId('description');
        expect(desc).toBeInTheDocument();
        expect(desc).toHaveClass('text-sm');
        expect(desc).toHaveClass('text-muted-foreground');
    });

    it('renders CardContent correctly', () => {
        render(<CardContent data-testid="content">Content goes here</CardContent>);
        const content = screen.getByTestId('content');
        expect(content).toBeInTheDocument();
        expect(content).toHaveClass('p-6');
    });

    it('renders CardFooter correctly', () => {
        render(<CardFooter data-testid="footer">Footer content</CardFooter>);
        const footer = screen.getByTestId('footer');
        expect(footer).toBeInTheDocument();
        expect(footer).toHaveClass('flex');
        expect(footer).toHaveClass('items-center');
    });

    it('applies custom className to Card', () => {
        render(<Card className="custom-card" data-testid="card">Content</Card>);
        expect(screen.getByTestId('card')).toHaveClass('custom-card');
    });

    it('applies custom className to CardHeader', () => {
        render(<CardHeader className="custom-header" data-testid="header">Header</CardHeader>);
        expect(screen.getByTestId('header')).toHaveClass('custom-header');
    });

    it('applies custom className to CardTitle', () => {
        render(<CardTitle className="custom-title">Title</CardTitle>);
        expect(screen.getByText('Title')).toHaveClass('custom-title');
    });

    it('applies custom className to CardDescription', () => {
        render(<CardDescription className="custom-desc">Desc</CardDescription>);
        expect(screen.getByText('Desc')).toHaveClass('custom-desc');
    });

    it('applies custom className to CardContent', () => {
        render(<CardContent className="custom-content">Content</CardContent>);
        expect(screen.getByText('Content')).toHaveClass('custom-content');
    });

    it('applies custom className to CardFooter', () => {
        render(<CardFooter className="custom-footer">Footer</CardFooter>);
        expect(screen.getByText('Footer')).toHaveClass('custom-footer');
    });

    it('renders Card without optional components', () => {
        render(<Card>Just content</Card>);
        expect(screen.getByText('Just content')).toBeInTheDocument();
    });

    it('forwards ref correctly for Card', () => {
        const ref = React.createRef<HTMLDivElement>();
        render(<Card ref={ref} data-testid="card">Content</Card>);
        expect(ref.current).toBeInTheDocument();
    });

    it('forwards ref correctly for CardHeader', () => {
        const ref = React.createRef<HTMLDivElement>();
        render(<CardHeader ref={ref} data-testid="header">Header</CardHeader>);
        expect(ref.current).toBeInTheDocument();
    });

    it('forwards ref correctly for CardContent', () => {
        const ref = React.createRef<HTMLDivElement>();
        render(<CardContent ref={ref} data-testid="content">Content</CardContent>);
        expect(ref.current).toBeInTheDocument();
    });

    it('forwards ref correctly for CardFooter', () => {
        const ref = React.createRef<HTMLDivElement>();
        render(<CardFooter ref={ref} data-testid="footer">Footer</CardFooter>);
        expect(ref.current).toBeInTheDocument();
    });
});
