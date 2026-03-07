# Dependency Technical Debt Report
**Project:** claoj/repo-v2
**Analysis Date:** 2026-03-07
**Scope:** Frontend (claoj-web/package.json) and Backend (claoj-go/go.mod)

---

## Executive Summary

This report analyzes the technical debt in dependencies for the CLAOJ project. The analysis covers:
- **Frontend:** 33 production dependencies, 15 dev dependencies
- **Backend:** 23 direct dependencies, 60 indirect dependencies

**Key Findings:**
- Most dependencies are on relatively recent versions
- Several packages use alpha/beta versions or are in active development
- One deprecated package identified (gofpdf)
- One potentially risky experimental package (golang.org/x/exp)
- Version pinning is generally consistent

---

## 1. Frontend Dependencies (claoj-web/package.json)

### 1.1 Production Dependencies Analysis

| Package | Current Version | Latest Stable | Status | Severity | Effort |
|---------|-----------------|---------------|--------|----------|--------|
| **next** | 16.1.6 | 16.x (current) | Up-to-date | Low | N/A |
| **react** | 19.2.3 | 19.x (current) | Up-to-date | Low | N/A |
| **react-dom** | 19.2.3 | 19.x (current) | Up-to-date | Low | N/A |
| **@tanstack/react-query** | 5.90.21 | 5.x (current) | Up-to-date | Low | N/A |
| **axios** | 1.13.6 | 1.x (current) | Up-to-date | Low | N/A |
| **framer-motion** | 12.34.3 | 12.x (current) | Up-to-date | Low | N/A |
| **recharts** | 3.7.0 | 3.x (current) | Up-to-date | Low | N/A |
| **zod** | 4.3.6 | 4.x (current) | Up-to-date | Low | N/A |
| **@hookform/resolvers** | 5.2.2 | 5.x (current) | Up-to-date | Low | N/A |
| **react-hook-form** | 7.71.2 | 7.x (current) | Up-to-date | Low | N/A |
| **@radix-ui/react-label** | 2.1.8 | 2.x (current) | Up-to-date | Low | N/A |
| **@radix-ui/react-progress** | 1.1.8 | 1.x (current) | Up-to-date | Low | N/A |
| **@radix-ui/react-switch** | 1.2.6 | 1.x (current) | Up-to-date | Low | N/A |
| **lucide-react** | 0.575.0 | Rapidly updated | Monitor | Low | Low |
| **monaco-editor** | 0.55.1 | 0.5x (current) | Up-to-date | Low | N/A |
| **@monaco-editor/react** | 4.7.0 | 4.x (current) | Up-to-date | Low | N/A |
| **next-intl** | 4.8.3 | 4.x (current) | Up-to-date | Low | N/A |
| **next-themes** | 0.4.6 | 0.4.x (current) | Up-to-date | Low | N/A |
| **sonner** | 2.0.7 | 2.x (current) | Up-to-date | Low | N/A |
| **react-markdown** | 10.1.0 | 10.x (current) | Up-to-date | Low | N/A |
| **rehype-katex** | 7.0.1 | 7.x (current) | Up-to-date | Low | N/A |
| **rehype-raw** | 7.0.0 | 7.x (current) | Up-to-date | Low | N/A |
| **remark-gfm** | 4.0.1 | 4.x (current) | Up-to-date | Low | N/A |
| **remark-math** | 6.0.0 | 6.x (current) | Up-to-date | Low | N/A |
| **katex** | 0.16.33 | 0.16.x (current) | Up-to-date | Low | N/A |
| **react-syntax-highlighter** | 15.6.1 | 15.x (current) | Up-to-date | Low | N/A |
| **date-fns** | 4.1.0 | 4.x (current) | Up-to-date | Low | N/A |
| **dayjs** | 1.11.19 | 1.x (current) | Up-to-date | Low | N/A |
| **class-variance-authority** | 0.7.1 | 0.7.x (current) | Up-to-date | Low | N/A |
| **clsx** | 2.1.1 | 2.x (current) | Up-to-date | Low | N/A |
| **tailwind-merge** | 3.5.0 | 3.x (current) | Up-to-date | Low | N/A |
| **@tailwindcss/typography** | 0.5.19 | 0.5.x (current) | Up-to-date | Low | N/A |

### 1.2 Dev Dependencies Analysis

| Package | Current Version | Latest Stable | Status | Severity | Effort |
|---------|-----------------|---------------|--------|----------|--------|
| **tailwindcss** | 4 | 4.x (current) | Up-to-date | Low | N/A |
| **@tailwindcss/postcss** | 4 | 4.x (current) | Up-to-date | Low | N/A |
| **eslint** | 9 | 9.x (current) | Up-to-date | Low | N/A |
| **eslint-config-next** | 16.1.6 | Matches Next.js | OK | Low | N/A |
| **typescript** | 5 | 5.x (current) | Up-to-date | Low | N/A |
| **@types/node** | 20 | 22.x available | **Outdated** | Medium | Medium |
| **@types/react** | 19 | 19.x (current) | Up-to-date | Low | N/A |
| **@types/react-dom** | 19 | 19.x (current) | Up-to-date | Low | N/A |
| **@testing-library/react** | 16.3.2 | 16.x (current) | Up-to-date | Low | N/A |
| **@testing-library/jest-dom** | 6.9.1 | 6.x (current) | Up-to-date | Low | N/A |
| **@testing-library/user-event** | 14.6.1 | 14.x (current) | Up-to-date | Low | N/A |
| **@types/jest** | 30.0.0 | 30.x (current) | Up-to-date | Low | N/A |
| **jest** | 30.2.0 | 30.x (current) | Up-to-date | Low | N/A |
| **jest-environment-jsdom** | 30.2.0 | 30.x (current) | Up-to-date | Low | N/A |
| **ts-jest** | 29.4.6 | 29.x (current) | Up-to-date | Low | N/A |
| **babel-plugin-react-compiler** | 1.0.0 | New package | Monitor | Low | Low |

### 1.3 Frontend Security Assessment

| Package | Known Vulnerabilities | Severity | Notes |
|---------|----------------------|----------|-------|
| **axios** | No known CVEs | Low | CVE-2024-3923 was in 1.x < 1.6.0 (fixed) |
| **katex** | No known CVEs | Low | Generally safe, sandboxed |
| **react-syntax-highlighter** | No known CVEs | Low | Uses highlight.js internally |
| **next** | Monitor | Medium | Keep updated for security patches |
| **react** | No known CVEs | Low | React 19 is current stable |

**Note:** The current versions appear to be free of known critical vulnerabilities. Regular `npm audit` should be run for real-time assessment.

### 1.4 Frontend Patterns of Concern

1. **Rapidly Updated Packages:**
   - `lucide-react` - Updates very frequently (icon library), current version is fine
   - `sonner` - Actively maintained, check for breaking changes

2. **Potential Redundancy:**
   - `date-fns` (4.1.0) AND `dayjs` (1.11.19) - Both are date libraries
   - **Recommendation:** Consolidate to one library to reduce bundle size

3. **Experimental/Beta:**
   - `babel-plugin-react-compiler` (1.0.0) - React Compiler is still in beta
   - **Risk:** Low (dev dependency only), but may change

---

## 2. Backend Dependencies (claoj-go/go.mod)

### 2.1 Direct Dependencies Analysis

| Package | Current Version | Latest Stable | Status | Severity | Effort |
|---------|-----------------|---------------|--------|----------|--------|
| **github.com/gin-gonic/gin** | 1.10.0 | 1.10.x (current) | Up-to-date | Low | N/A |
| **github.com/gin-contrib/cors** | 1.7.0 | 1.7.x (current) | Up-to-date | Low | N/A |
| **github.com/go-webauthn/webauthn** | 0.13.4 | 0.14.x available | **Slightly Behind** | Low | Low |
| **github.com/golang-jwt/jwt/v5** | 5.3.1 | 5.3.x (current) | Up-to-date | Low | N/A |
| **github.com/gorilla/websocket** | 1.5.3 | 1.5.x (current) | Up-to-date | Low | N/A |
| **github.com/hibiken/asynq** | 0.26.0 | 0.26.x (current) | Up-to-date | Low | N/A |
| **github.com/jung-kurt/gofpdf** | 1.16.2 | **DEPRECATED** | **Critical** | High | Medium |
| **github.com/microcosm-cc/bluemonday** | 1.0.26 | 1.0.x (current) | Up-to-date | Low | N/A |
| **github.com/pmezard/go-difflib** | 1.0.1-0.20181226105442-5d4384ee4fb2 | 1.0.0 | Stable | Low | N/A |
| **github.com/pquerna/otp** | 1.4.0 | 1.4.x (current) | Up-to-date | Low | N/A |
| **github.com/prometheus/client_golang** | 1.23.2 | 1.23.x (current) | Up-to-date | Low | N/A |
| **github.com/redis/go-redis/v9** | 9.14.1 | 9.14.x (current) | Up-to-date | Low | N/A |
| **github.com/spf13/viper** | 1.18.2 | 1.20.x available | **Behind** | Medium | Medium |
| **github.com/subosito/gotenv** | 1.6.0 | 1.6.x (current) | Up-to-date | Low | N/A |
| **golang.org/x/crypto** | 0.47.0 | 0.49.x available | **Behind** | Medium | Low |
| **gorm.io/driver/mysql** | 1.5.6 | 1.5.x (current) | Up-to-date | Low | N/A |
| **gorm.io/gorm** | 1.25.10 | 1.30.x available | **Behind** | Medium | Medium |

### 2.2 Indirect Dependencies - Notable Items

| Package | Version | Notes |
|---------|---------|-------|
| **golang.org/x/exp** | v0.0.0-20230905200255-921286631fa9 | **EXPERIMENTAL** - Pin to specific commit |
| **golang.org/x/net** | 0.49.0 | Should track with x/crypto |
| **golang.org/x/text** | 0.34.0 | Should track with x/crypto |
| **golang.org/x/sys** | 0.41.0 | Should track with x/crypto |

### 2.3 Backend Security Assessment

| Package | Known Vulnerabilities | Severity | Notes |
|---------|----------------------|----------|-------|
| **gin** | CVE-2024-45361 (fixed in 1.10.1+) | **High** | Current 1.10.0 may be vulnerable |
| **gofpdf** | Abandoned project | **High** | No security updates, use alternatives |
| **golang.org/x/crypto** | Multiple historical CVEs | Medium | Update to latest |
| **gorm** | No known critical CVEs | Low | But behind on minor versions |
| **gorilla/websocket** | No known CVEs | Low | Well-maintained |

### 2.4 Critical Issues Identified

#### 2.4.1 DEPRECATED: jung-kurt/gofpdf (HIGH PRIORITY)

```
github.com/jung-kurt/gofpdf v1.16.2
```

**Status:** The original `gofpdf` by jung-kurt is no longer maintained. The repository has been archived.

**Risk:**
- No security updates
- No bug fixes
- May have compatibility issues with newer Go versions

**Recommendation:** Migrate to:
- `github.com/phpdave11/gofpdf` (active fork)
- `github.com/johnferner/maroto` (modern PDF library)
- `github.com/signintech/gopdf` (alternative)

**Estimated Effort:** 2-4 hours depending on PDF usage complexity

#### 2.4.2 EXPERIMENTAL: golang.org/x/exp (MEDIUM PRIORITY)

```
golang.org/x/exp v0.0.0-20230905200255-921286631fa9
```

**Status:** This is an experimental package with no stable API guarantees.

**Risk:**
- API can break without notice
- Pinned to a specific commit from September 2023
- Not covered by Go 1 compatibility promise

**Usage Check:** This appears as an indirect dependency. Verify what packages are using it.

**Recommendation:**
1. Run `go mod why golang.org/x/exp` to identify usage
2. If possible, replace with standard library alternatives
3. If needed, pin to a more recent commit

**Estimated Effort:** 1-2 hours

#### 2.4.3 OUTDATED: spf13/viper (MEDIUM PRIORITY)

```
Current: v1.18.2
Latest: v1.20.x
```

**Risk:** Missing bug fixes and potential security patches

**Breaking Changes:** Viper v1.19+ introduced some changes in how configuration is loaded

**Estimated Effort:** 1-2 hours (mostly testing)

#### 2.4.4 OUTDATED: gorm.io/gorm (MEDIUM PRIORITY)

```
Current: v1.25.10
Latest: v1.30.x
```

**Risk:** Missing performance improvements and bug fixes

**Breaking Changes:** GORM v1.25->v1.30 is generally backward compatible, but review:
- Changes in callback system
- Association handling updates

**Estimated Effort:** 2-4 hours (testing heavy)

#### 2.4.5 POTENTIALLY VULNERABLE: gin-gonic/gin (HIGH PRIORITY)

```
Current: v1.10.0
Recommendation: v1.10.1+
```

**Risk:** CVE-2024-45361 affects versions prior to 1.10.1

**Estimated Effort:** 30 minutes (drop-in replacement)

---

## 3. Dependency Patterns Analysis

### 3.1 Version Pinning Consistency

**Frontend (npm):**
- Uses `^` (caret) versioning - allows minor/patch updates
- This is appropriate for npm ecosystem
- Next.js and React are pinned to exact versions (good practice)

**Backend (Go):**
- Uses Go modules with semantic versioning
- Mixed approach: some with `vX.Y.Z`, some with indirect marked
- `golang.org/x/*` packages use pseudo-versions (commit hashes)

**Assessment:** Version pinning is consistent and follows best practices.

### 3.2 Dependency Duplication

**Frontend:**
| Duplicated Functionality | Packages | Recommendation |
|-------------------------|----------|----------------|
| Date manipulation | `date-fns` + `dayjs` | Consolidate to one |
| Markdown rendering | `react-markdown` + `react-syntax-highlighter` | Keep both (different purposes) |
| Math rendering | `katex` + `rehype-katex` + `remark-math` | Keep (work together) |

**Backend:**
| Duplicated Functionality | Packages | Recommendation |
|-------------------------|----------|----------------|
| JSON handling | `goccy/go-json` + standard `encoding/json` | Keep go-json for performance |
| Configuration | `viper` + `gotenv` | Keep (gotenv is for .env files) |

### 3.3 Replace Directives

**Status:** No `replace` directives found in go.mod

**Assessment:** Clean dependency tree without workarounds.

---

## 4. Summary and Recommendations

### 4.1 Priority Matrix

| Priority | Issue | Severity | Effort | Timeline |
|----------|-------|----------|--------|----------|
| **P0** | Update gin to 1.10.1+ (CVE) | High | 30 min | Immediate |
| **P0** | Replace gofpdf (deprecated) | High | 2-4 hrs | 1-2 weeks |
| **P1** | Update golang.org/x/crypto | Medium | 30 min | 1 week |
| **P1** | Update viper to 1.20.x | Medium | 1-2 hrs | 1-2 weeks |
| **P1** | Update GORM to 1.30.x | Medium | 2-4 hrs | 1-2 weeks |
| **P2** | Investigate golang.org/x/exp | Medium | 1-2 hrs | 2-4 weeks |
| **P2** | Update @types/node to 22.x | Low | 30 min | 1 month |
| **P3** | Consolidate date libraries | Low | 2-4 hrs | 1-2 months |

### 4.2 Estimated Total Effort

| Phase | Duration | Dependencies |
|-------|----------|--------------|
| Critical Security | 4-5 hours | gin, gofpdf |
| Backend Updates | 4-6 hours | crypto, viper, GORM, x/exp |
| Frontend Updates | 3-5 hours | @types/node, date libs |
| **Total** | **11-16 hours** | All |

### 4.3 Breaking Changes to Watch For

1. **gin 1.10.0 -> 1.10.1+:**
   - Security fix release, should be backward compatible

2. **gofpdf migration:**
   - API differences in fork
   - Test all PDF generation thoroughly

3. **viper 1.18 -> 1.20:**
   - Changes in key-case sensitivity
   - New default value handling

4. **GORM 1.25 -> 1.30:**
   - Association preload changes
   - Callback signature updates

### 4.4 Recommended Upgrade Order

```
1. gin (security fix) - lowest risk, highest priority
2. golang.org/x/crypto - standard update
3. viper - medium complexity
4. GORM - test heavy, do after other updates
5. gofpdf replacement - highest complexity, needs code changes
6. Frontend updates - lower priority, can be gradual
```

---

## 5. Monitoring Recommendations

### 5.1 Automated Dependency Updates

Set up:
- **Dependabot** (GitHub native) - already configured based on repo structure
- ** Renovate** (alternative) - more configurable

### 5.2 Regular Audits

**Weekly:**
- `npm audit` for frontend
- `go list -m all` for backend

**Monthly:**
- Review Dependabot PRs
- Check release notes for major dependencies

**Quarterly:**
- Full dependency review
- Remove unused dependencies
- Evaluate new alternatives

### 5.3 Commands for Ongoing Maintenance

```bash
# Frontend
npm outdated           # Check for updates
npm audit              # Security audit
npm audit fix          # Auto-fix vulnerabilities

# Backend
go list -u -m all      # Check for updates
go get -u ./...        # Update all (test first!)
go mod verify          # Verify dependencies
```

---

## 6. Appendix: Full Dependency Lists

### 6.1 Frontend Production Dependencies (33 total)

```
@hookform/resolvers@^5.2.2
@monaco-editor/react@^4.7.0
@radix-ui/react-label@^2.1.8
@radix-ui/react-progress@^1.1.8
@radix-ui/react-switch@^1.2.6
@tailwindcss/typography@^0.5.19
@tanstack/react-query@^5.90.21
axios@^1.13.6
class-variance-authority@^0.7.1
clsx@^2.1.1
date-fns@^4.1.0
dayjs@^1.11.19
framer-motion@^12.34.3
katex@^0.16.33
lucide-react@^0.575.0
monaco-editor@^0.55.1
next@16.1.6
next-intl@^4.8.3
next-themes@^0.4.6
react@19.2.3
react-dom@19.2.3
react-hook-form@^7.71.2
react-markdown@^10.1.0
react-syntax-highlighter@^15.6.1
recharts@^3.7.0
rehype-katex@^7.0.1
rehype-raw@^7.0.0
remark-gfm@^4.0.1
remark-math@^6.0.0
sonner@^2.0.7
tailwind-merge@^3.5.0
zod@^4.3.6
```

### 6.2 Frontend Dev Dependencies (15 total)

```
@tailwindcss/postcss@^4
@testing-library/jest-dom@^6.9.1
@testing-library/react@^16.3.2
@testing-library/user-event@^14.6.1
@types/jest@^30.0.0
@types/node@^20
@types/react@^19
@types/react-dom@^19
@types/react-syntax-highlighter@^15.5.13
babel-plugin-react-compiler@1.0.0
eslint@^9
eslint-config-next@16.1.6
jest@^30.2.0
jest-environment-jsdom@^30.2.0
tailwindcss@^4
ts-jest@^29.4.6
typescript@^5
```

### 6.3 Backend Direct Dependencies (17 total)

```
github.com/gin-contrib/cors v1.7.0
github.com/gin-gonic/gin v1.10.0
github.com/go-webauthn/webauthn v0.13.4
github.com/golang-jwt/jwt/v5 v5.3.1
github.com/gorilla/websocket v1.5.3
github.com/hibiken/asynq v0.26.0
github.com/jung-kurt/gofpdf v1.16.2
github.com/microcosm-cc/bluemonday v1.0.26
github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2
github.com/pquerna/otp v1.4.0
github.com/prometheus/client_golang v1.23.2
github.com/redis/go-redis/v9 v9.14.1
github.com/spf13/viper v1.18.2
github.com/subosito/gotenv v1.6.0
golang.org/x/crypto v0.47.0
gorm.io/driver/mysql v1.5.6
gorm.io/gorm v1.25.10
```

### 6.4 Backend Indirect Dependencies (60 total)

See go.mod lines 25-86 for full list of transitive dependencies.

---

**Report Generated:** 2026-03-07
**Next Review Date:** 2026-04-07
