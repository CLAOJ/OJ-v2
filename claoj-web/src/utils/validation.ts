import { z } from 'zod';

// Maximum lengths
const MAX_COMMENT_LENGTH = 10000;
const MAX_USERNAME_LENGTH = 30;
const MAX_PASSWORD_LENGTH = 128;
const MIN_USERNAME_LENGTH = 3;
const MIN_PASSWORD_LENGTH = 8;

// Username regex: alphanumeric and underscore only
const USERNAME_REGEX = /^[a-zA-Z0-9_]+$/;

// Password strength regex
const PASSWORD_STRENGTH_REGEX = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]/;

/**
 * Sanitizes HTML content in comments by removing dangerous tags
 * This is a client-side safeguard; server-side sanitization is primary
 */
export function sanitizeComment(content: string): string {
    if (!content) return '';

    // Truncate if too long
    let sanitized = content.slice(0, MAX_COMMENT_LENGTH);

    // Remove script tags and their content
    sanitized = sanitized.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '');

    // Remove event handlers
    sanitized = sanitized.replace(/\son\w+\s*=\s*["'][^"']*["']/gi, '');

    // Remove style tags
    sanitized = sanitized.replace(/<style\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/style>/gi, '');

    return sanitized;
}

/**
 * Validates username according to CLAOJ rules
 * - 3-30 characters
 * - Alphanumeric and underscore only
 */
export function validateUsername(username: string): boolean {
    if (!username) return false;
    if (username.length < MIN_USERNAME_LENGTH || username.length > MAX_USERNAME_LENGTH) {
        return false;
    }
    return USERNAME_REGEX.test(username);
}

/**
 * Validates password strength
 * - At least 8 characters
 * - Contains uppercase, lowercase, number, and special character
 */
export function validatePassword(password: string): boolean {
    if (!password) return false;
    if (password.length < MIN_PASSWORD_LENGTH || password.length > MAX_PASSWORD_LENGTH) {
        return false;
    }
    return PASSWORD_STRENGTH_REGEX.test(password);
}

/**
 * Validates email format
 */
export function validateEmail(email: string): boolean {
    if (!email) return false;
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
}

/**
 * Zod schema for registration validation
 */
export const registerSchema = z.object({
    username: z
        .string()
        .min(MIN_USERNAME_LENGTH, 'Username must be at least 3 characters')
        .max(MAX_USERNAME_LENGTH, 'Username must be at most 30 characters')
        .regex(USERNAME_REGEX, 'Username can only contain letters, numbers, and underscores'),
    email: z.string().email('Invalid email address'),
    password: z
        .string()
        .min(MIN_PASSWORD_LENGTH, 'Password must be at least 8 characters')
        .max(MAX_PASSWORD_LENGTH, 'Password is too long')
        .regex(PASSWORD_STRENGTH_REGEX, 'Password must contain uppercase, lowercase, number, and special character'),
    confirmPassword: z.string(),
}).refine((data) => data.password === data.confirmPassword, {
    message: "Passwords don't match",
    path: ['confirmPassword'],
});

/**
 * Zod schema for login validation
 */
export const loginSchema = z.object({
    username: z.string().min(1, 'Username is required'),
    password: z.string().min(1, 'Password is required'),
});
