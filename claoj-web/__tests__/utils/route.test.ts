import { describe, it, expect } from '@jest/globals';
import { parseLeadingId } from '@/utils/route';

describe('parseLeadingId', () => {
  it('extracts the numeric id from an <id>-<slug> segment', () => {
    expect(parseLeadingId('93-lấytiền')).toBe('93');
  });
  it('handles a slug containing extra hyphens', () => {
    expect(parseLeadingId('1-thpt-dh')).toBe('1');
  });
  it('returns the whole segment when there is no slug', () => {
    expect(parseLeadingId('42')).toBe('42');
  });
  it('returns an empty string for an empty segment', () => {
    expect(parseLeadingId('')).toBe('');
  });
});
