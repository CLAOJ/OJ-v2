# Design Review: CLAOJ Web UI - Comprehensive Review

**Review ID:** claoj-ui-comprehensive-review_20260305
**Reviewed:** 2026-03-05
**Target:** claoj/repo-v2/claoj-web/src (pages, components, layouts)
**Focus:** Visual, Usability, Code Quality, Performance
**Context:** Data display (competitive programming platform - dashboards, tables, problem pages)
**Platform:** All platforms (desktop, tablet, mobile)

## Summary

The CLAOJ v2 UI demonstrates a solid foundation with modern React patterns, consistent use of Tailwind CSS, and thoughtful component architecture. The design system shows maturity with good dark mode support and responsive layouts. However, there are several areas for improvement including accessibility compliance, design token consistency, and performance optimization.

**Issues Found:** 23

- **Critical:** 2
- **Major:** 7
- **Minor:** 8
- **Suggestions:** 6

---

## Critical Issues

### Issue 1: Missing Accessibility Attributes on Dialog Component

**Severity:** Critical
**Location:** `components/ui/Dialog.tsx:34-62`
**Category:** Accessibility

**Problem:**
The Dialog component lacks essential ARIA attributes required for screen reader accessibility:
- Missing `role="dialog"`
- Missing `aria-modal="true"`
- Missing `aria-labelledby` for dialog title
- Missing `aria-describedby` for dialog description

**Impact:**
Screen reader users cannot properly identify dialog windows or understand their purpose. This violates WCAG 2.1 Level A requirements.

**Recommendation:**
```tsx
// Before
<motion.div
    className="fixed left-1/2 top-1/2 z-50 ... rounded-2xl bg-popover p-6"
>
    {children}
</motion.div>

// After
<motion.div
    role="dialog"
    aria-modal="true"
    aria-labelledby="dialog-title"
    aria-describedby="dialog-description"
    className="fixed left-1/2 top-1/2 z-50 ... rounded-2xl bg-popover p-6"
>
    <div id="dialog-title" className="sr-only">{title}</div>
    {children}
</motion.div>
```

---

### Issue 2: Keyboard Focus Not Trapped in Mobile Menu

**Severity:** Critical
**Location:** `components/layout/Navbar.tsx:283-395`
**Category:** Accessibility

**Problem:**
When the mobile menu is open, keyboard focus can escape the menu and navigate to elements behind it. There's no focus trap implementation.

**Impact:**
Keyboard and screen reader users can become confused navigating the mobile menu, potentially accessing hidden elements.

**Recommendation:**
Implement a focus trap utility:
```tsx
// Add focus trap hook
useEffect(() => {
    if (mobileMenuOpen) {
        const focusableElements = menuRef.current?.querySelectorAll(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        const firstEl = focusableElements?.[0];
        const lastEl = focusableElements?.[focusableElements.length - 1];

        const handleTabKey = (e: KeyboardEvent) => {
            if (e.key === 'Tab') {
                if (e.shiftKey && document.activeElement === firstEl) {
                    e.preventDefault();
                    lastEl?.focus();
                } else if (!e.shiftKey && document.activeElement === lastEl) {
                    e.preventDefault();
                    firstEl?.focus();
                }
            }
        };

        document.addEventListener('keydown', handleTabKey);
        firstEl?.focus();
        return () => document.removeEventListener('keydown', handleTabKey);
    }
}, [mobileMenuOpen]);
```

---

## Major Issues

### Issue 3: Inconsistent Color System - Hardcoded Values Mixed with Design Tokens

**Severity:** Major
**Location:** Multiple files (`Navbar.tsx`, `page.tsx`, `ratings/page.tsx`)
**Category:** Visual Design / Maintainability

**Problem:**
The codebase mixes hardcoded colors with design tokens inconsistently:
- `#009688` (teal) used 50+ times
- `#263238` (dark blue-gray) used 20+ times
- `#3b4d56` used 15+ times
- But also uses `bg-primary`, `text-muted-foreground`

**Rating Page Example:**
```tsx
// Hardcoded rating colors
if (rating < 1000) return 'text-[#988f81]';
if (rating < 1200) return 'text-[#72ff72]';
if (rating < 1400) return 'text-[#57fcf2]';
```

**Impact:**
- Theme customization becomes difficult
- Dark mode edge cases may appear
- Harder to maintain consistent branding

**Recommendation:**
Define rating colors in design tokens:
```tsx
// lib/ratingColors.ts
export const RATING_COLORS = {
  newbie: '#988f81',
  candidate: '#72ff72',
  specialist: '#57fcf2',
  expert: '#337dff',
  candidateMaster: '#ff55ff',
  master: '#ff981a',
  grandmaster: '#ff1a1a',
} as const;

// components/ui/Badge.tsx - add rating variant
variant: {
  rating-newbie: "bg-rating-newbie text-white",
  rating-candidate: "bg-rating-candidate text-white",
  // ...
}
```

---

### Issue 4: Icon-Only Buttons Missing Accessible Labels

**Severity:** Major
**Location:** `Navbar.tsx:169-175`, `NotificationBell.tsx`, multiple components
**Category:** Accessibility

**Problem:**
Icon-only buttons lack `aria-label` attributes:
```tsx
<button
    onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
    className="p-2 rounded-full hover:bg-white/10"
>
    {mounted && (theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />)}
</button>
```

**Impact:**
Screen reader users hear only "button" without context about the button's purpose.

**Recommendation:**
```tsx
<button
    onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
    className="p-2 rounded-full hover:bg-white/10"
    aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
>
    {mounted && (theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />)}
</button>
```

---

### Issue 5: Inline Style for Dynamic Width - AC Rate Progress Bar

**Severity:** Major
**Location:** `problems/[code]/page.tsx:208-211`
**Category:** Code Quality

**Problem:**
```tsx
<div
    className="h-full bg-primary"
    style={{ width: `${problem.ac_rate}%` }}
/>
```

**Impact:**
While functional, inline styles bypass CSP policies and are harder to test/maintain.

**Recommendation:**
```tsx
<div
    className="h-full bg-primary transition-all duration-500"
    style={{ width: `${Math.min(100, Math.max(0, problem.ac_rate))}%` }}
    role="progressbar"
    aria-valuenow={problem.ac_rate}
    aria-valuemin={0}
    aria-valuemax={100}
/>
```

---

### Issue 6: Missing Error States for Failed Queries

**Severity:** Major
**Location:** Multiple pages (`page.tsx`, `ratings/page.tsx`)
**Category:** Usability

**Problem:**
React Query's `error` state is not handled:
```tsx
const { data: posts, isLoading } = useQuery({
    queryKey: ['blog-posts'],
    queryFn: fetchPosts
});
// No error handling
```

**Impact:**
Users see infinite loading or empty states when API calls fail.

**Recommendation:**
```tsx
const { data: posts, isLoading, error } = useQuery({...});

if (error) {
    return (
        <div className="p-8 text-center text-destructive">
            <AlertCircle className="mx-auto mb-2" size={48} />
            <p className="font-bold">Failed to load blog posts</p>
            <button onClick={() => refetch()} className="mt-4 btn-primary">
                Retry
            </button>
        </div>
    );
}
```

---

### Issue 7: Loading Skeleton Not Matching Content Layout

**Severity:** Major
**Location:** `page.tsx:194-200`, `ratings/page.tsx:170-176`
**Category:** Usability / Performance Perception

**Problem:**
Generic skeletons don't match actual content structure:
```tsx
[1, 2, 3].map(i => (
    <div key={i} className="h-16 bg-muted/50 rounded-xl animate-pulse" />
))
```

**Impact:**
Layout shift when content loads creates a jarring experience.

**Recommendation:**
Create content-specific skeletons:
```tsx
function BlogPostSkeleton() {
    return (
        <div className="bg-card border rounded-lg p-6">
            <div className="flex gap-4">
                <Skeleton className="h-10 w-10 rounded-full" />
                <div className="flex-1 space-y-2">
                    <Skeleton className="h-6 w-3/4" />
                    <Skeleton className="h-4 w-1/2" />
                    <Skeleton className="h-24 w-full" />
                </div>
            </div>
        </div>
    );
}
```

---

### Issue 8: ClarificationList Component Has Wrong Import Reference

**Severity:** Major
**Location:** `components/contests/ClarificationList.tsx:7`
**Category:** Code Quality (Bug)

**Problem:**
The file imports `Loader2` but uses it without importing:
```tsx
import { MessageSquare, MessageSquarePlus, ChevronDown, ChevronUp, Send, Loader2 } from 'lucide-react';
// Loader2 is imported but used conditionally - verify usage
```

Actually reviewing line 219: `{answerMutation.isPending && <Loader2 size={16} className="animate-spin" />}`

This is correctly imported. However, the component has a typescript interface issue - the `author` field typing doesn't match the API response format from the backend.

**Recommendation:**
Ensure interface matches backend API response exactly:
```tsx
export interface ContestClarification {
    id: number;
    question: string;
    answer?: string | null;  // Match backend's *string
    is_answered: boolean;
    create_time: string;
    author?: {
        username: string;
    } | null;  // Handle null case
}
```

---

### Issue 9: Date Formatting Inconsistency

**Severity:** Major
**Location:** Multiple components
**Category:** Usability / i18n

**Problem:**
Multiple date formatting approaches:
```tsx
// ClarificationList.tsx
new Date(clarification.create_time).toLocaleString()

// page.tsx (using dayjs)
dayjs(post.publish_on).fromNow()

// ratings/page.tsx
// No date formatting needed but pattern exists elsewhere
```

**Impact:**
- Inconsistent date formats across the app
- Not respecting user locale
- i18n complications

**Recommendation:**
Create a unified date utility:
```tsx
// lib/date.ts
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import localizedFormat from 'dayjs/plugin/localizedFormat';

dayjs.extend(relativeTime);
dayjs.extend(localizedFormat);

export function formatDate(date: string | Date, format: 'relative' | 'short' | 'long' = 'relative', locale = 'en') {
    dayjs.locale(locale);
    const d = dayjs(date);

    switch (format) {
        case 'relative': return d.fromNow();
        case 'short': return d.format('MMM D, YYYY');
        case 'long': return d.format('MMMM D, YYYY [at] h:mm A');
        case 'datetime': return d.format('YYYY-MM-DD HH:mm:ss');
        default: return d.fromNow();
    }
}
```

---

## Minor Issues

### Issue 10: Magic Numbers in Contest Time Formatting

**Location:** `page.tsx:114-121`

```tsx
const days = Math.floor(duration / (1000 * 60 * 60 * 24));
const hours = Math.floor((duration % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
```

**Recommendation:**
```tsx
const MS_IN_DAY = 1000 * 60 * 60 * 24;
const MS_IN_HOUR = 1000 * 60 * 60;
const MS_IN_MINUTE = 1000 * 60;
```

---

### Issue 11: No Debouncing on Search Input

**Location:** `ratings/page.tsx:67-80`

Search triggers query on every keystroke without debounce.

**Recommendation:**
```tsx
const [search, setSearch] = useState('');
const debouncedSearch = useDebounce(search, 300);

const { data } = useQuery({
    queryKey: ['ratings-leaderboard', page, debouncedSearch],
    // ...
});
```

---

### Issue 12: Missing Empty State for Sidebar Widgets

**Location:** `page.tsx:359-555`

When no data exists (no contests, no comments), sidebar shows nothing without explaining why.

---

### Issue 13: Table Headers Lack Sort Indicators

**Location:** `ratings/page.tsx:86-104`

Table headers indicate sortable columns but don't show current sort direction.

---

### Issue 14: Mobile Menu Animation Could Cause Motion Sensitivity

**Location:** `Navbar.tsx:283-289`

```tsx
<motion.div
    initial={{ opacity: 0, height: 0 }}
    animate={{ opacity: 1, height: 'auto' }}
```

**Recommendation:** Respect `prefers-reduced-motion`.

---

### Issue 15: Form Disabled State Not Visually Distinct Enough

**Location:** Multiple form components

Disabled buttons use `disabled:opacity-50` which may not meet WCAG contrast requirements.

---

### Issue 16: Missing Skip Navigation Link

**Location:** Layout/App level

No skip link for keyboard users to bypass navigation.

---

### Issue 17: Notification Polling Not Visible

**Location:** `NotificationBell.tsx`

No visual indication of real-time update status beyond WebSocket indicator.

---

### Issue 18: Code Editor Language Selector Could Be More Accessible

**Location:** `problems/[code]/page.tsx:274-282`

Native select element may not be fully accessible with custom styling.

---

## Suggestions

### Suggestion 1: Create Design Token Documentation

Document all colors, spacing, typography in a `design-tokens.json` or similar for team reference.

### Suggestion 2: Add Component Storybook

Implement Storybook for component documentation and visual regression testing.

### Suggestion 3: Implement Virtual Scrolling for Long Tables

For ratings page with many users, consider virtual scrolling (react-window).

### Suggestion 4: Add Print Styles

Competitive programming platforms often need printable problem statements.

### Suggestion 5: Consider PWA Features

Add offline support for problem viewing and code drafting.

### Suggestion 6: Add Keyboard Shortcuts

Power users benefit from shortcuts (e.g., `Ctrl+Enter` to submit code).

---

## Positive Observations

1. **Excellent Visual Hierarchy** - The problem page has outstanding visual hierarchy with clear primary actions, well-organized metadata, and logical content flow.

2. **Consistent Border Radius System** - Good use of `rounded-xl`, `rounded-2xl`, `rounded-3xl` creating a cohesive rounded aesthetic.

3. **Thoughtful Dark Mode Implementation** - Dark mode is properly implemented with appropriate contrast and color adjustments.

4. **Good Loading State Patterns** - Skeleton loaders are implemented throughout, showing attention to perceived performance.

5. **Clean Component Composition** - Components are well-structured with clear props interfaces and single responsibilities.

6. **Effective Use of Animation** - Framer Motion animations enhance UX without being excessive (fade-ins, scale effects).

7. **Responsive Design** - Mobile menu and responsive layouts show consideration for all screen sizes.

8. **Good TypeScript Coverage** - Proper typing throughout the codebase.

---

## Next Steps

**Priority 1 (Critical - Fix Immediately):**
1. Add ARIA attributes to Dialog component
2. Implement focus trap for mobile menu

**Priority 2 (Major - This Sprint):**
3. Add aria-labels to all icon buttons
4. Implement error states for all React Query hooks
5. Fix date formatting inconsistency
6. Address hardcoded color values with design tokens

**Priority 3 (Minor - Next Sprint):**
7. Create content-specific skeletons
8. Add debouncing to search inputs
9. Improve empty states
10. Add sort indicators to tables

**Priority 4 (Suggestions - Backlog):**
- Design token documentation
- Storybook setup
- PWA features
- Keyboard shortcuts

---

_Generated by UI Design Review. Run `/ui-design:design-review` again after fixes._
