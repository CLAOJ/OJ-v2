import { describe, it, expect } from '@jest/globals';
import { sanitizeComment, validateUsername, validatePassword } from '@/utils/validation';

describe('Validation Utilities', () => {
  describe('sanitizeComment', () => {
    it('should remove script tags', () => {
      const input = '<script>alert("xss")</script>Hello';
      const result = sanitizeComment(input);
      expect(result).not.toContain('<script>');
    });

    it('should allow safe HTML', () => {
      const input = '<p>Hello <strong>world</strong></p>';
      const result = sanitizeComment(input);
      expect(result).toContain('Hello');
    });

    it('should truncate long comments', () => {
      const input = 'a'.repeat(15000);
      const result = sanitizeComment(input);
      expect(result.length).toBeLessThanOrEqual(10000);
    });
  });

  describe('validateUsername', () => {
    it('should accept valid usernames', () => {
      expect(validateUsername('validuser')).toBe(true);
      expect(validateUsername('user_123')).toBe(true);
    });

    it('should reject short usernames', () => {
      expect(validateUsername('ab')).toBe(false);
    });

    it('should reject invalid characters', () => {
      expect(validateUsername('user@name')).toBe(false);
    });
  });

  describe('validatePassword', () => {
    it('should accept strong passwords', () => {
      expect(validatePassword('SecurePass123!')).toBe(true);
    });

    it('should reject short passwords', () => {
      expect(validatePassword('abc')).toBe(false);
    });

    it('should reject weak passwords', () => {
      expect(validatePassword('onlylowercase')).toBe(false);
    });
  });
});
