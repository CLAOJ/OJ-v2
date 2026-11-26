import { isStockSource, resolveLanguage, templateFor } from '@/lib/codeLanguage';

describe('codeLanguage', () => {
    it('maps DMOJ keys to highlighter ids', () => {
        expect(resolveLanguage('CPP17')).toBe('cpp');
        expect(resolveLanguage('CPP23')).toBe('cpp');
        expect(resolveLanguage('PY3')).toBe('python');
        expect(resolveLanguage('C')).toBe('c');
        expect(resolveLanguage('')).toBe('text');
    });

    it('picks a per-language template and only treats stock code as replaceable', () => {
        expect(templateFor('PY3')).toContain('def main');
        expect(templateFor('CPP17')).toContain('#include');
        expect(templateFor('UNKNOWN')).toBe('');
        expect(isStockSource('')).toBe(true);
        expect(isStockSource(templateFor('JAVA17'))).toBe(true);
        expect(isStockSource('print(1)')).toBe(false);
    });
});
