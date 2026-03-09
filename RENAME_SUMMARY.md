# Repository Rename Summary

## Changes Made

### 1. Directory Structure

**Before:**
```
claoj/
├── repo-v2/
│   ├── claoj-go/      # Backend
│   └── claoj-web/     # Frontend
└── claoj-judge-go/    # Standalone judge (created earlier)
```

**After:**
```
claoj/
└── repo/
    ├── claoj/         # Backend (renamed from claoj-go)
    ├── claoj-judge/   # Standalone Go judge (moved from claoj-judge-go)
    └── claoj-web/     # Frontend
```

### 2. Module Path Changes

**Backend (claoj):**
- Old: `github.com/CLAOJ/claoj-go`
- New: `github.com/CLAOJ/claoj`

**Judge (claoj-judge):**
- Old: `github.com/CLAOJ/claoj-judge-go`
- New: `github.com/CLAOJ/claoj-judge`

### 3. API Endpoint Changes

All API endpoints changed from `/api/v2` to `/api`:

**Examples:**
- `POST /api/v2/auth/login` → `POST /api/auth/login`
- `GET /api/v2/problems` → `GET /api/problems`
- `POST /api/v2/problem/:code/submit` → `POST /api/problem/:code/submit`
- `GET /api/v2/admin/problems` → `GET /api/admin/problems`

### 4. Files Modified

**Backend:**
- `repo/claoj/go.mod` - Module path updated
- `repo/claoj/main.go` - Import paths updated
- `repo/claoj/api/router.go` - Route prefix changed from `/api/v2` to `/api`
- All `*.go` files - Import paths updated via sed

**Judge:**
- `repo/claoj-judge/go.mod` - Module path updated
- `repo/claoj-judge/cmd/claoj-judge/main.go` - Import paths updated
- `repo/claoj-judge/core/*.go` - Import paths updated
- `repo/claoj-judge/protocol/packet.go` - Import paths updated
- `repo/claoj-judge/config/config.go` - No changes needed
- `repo/claoj-judge/executors/*.go` - No import changes needed

### 5. Frontend Changes Needed

The frontend (claoj-web) will need to be updated to use the new API endpoints:

**Files to update:**
- `claoj-web/src/lib/api/*.ts` - API client files
- `claoj-web/src/i18n/*.json` - If any API paths are hardcoded
- Any component files that make direct API calls

**Example changes:**
```typescript
// Before
const API_BASE = '/api/v2';

// After
const API_BASE = '/api';
```

## Testing Checklist

### Backend
- [ ] `go build ./...` compiles without errors
- [ ] `go test ./...` passes all tests
- [ ] Backend starts and listens on port 8080
- [ ] Health endpoint works: `curl http://localhost:8080/health`

### Judge
- [ ] `go build ./...` compiles without errors
- [ ] `go test ./...` passes all tests
- [ ] Docker image builds: `docker build -t claoj/judge-go:latest .`
- [ ] Judges can connect to backend

### Integration
- [ ] Backend bridge listens on port 9999
- [ ] Judges can handshake with backend
- [ ] Submissions can be graded

## Migration Commands

### Update frontend API calls (example)
```bash
cd claoj/repo/claoj-web
# Find all /api/v2 references
grep -r "/api/v2" src/

# Replace with /api
sed -i 's|/api/v2|/api|g' src/**/*.ts
```

### Update database (if API paths are stored)
```sql
-- No database changes needed for API path change
-- The route prefix is handled in code
```

## Rollback Plan

If issues occur, revert the changes:

```bash
# Revert router.go change
# Change /api back to /api/v2 in repo/claoj/api/router.go

# Revert module paths
# Change github.com/CLAOJ/claoj back to github.com/CLAOJ/claoj-go
```

## Next Steps

1. Update frontend code to use `/api` instead of `/api/v2`
2. Test the complete flow: frontend → backend → judge
3. Update any documentation that references `/api/v2`
4. Deploy to staging environment
5. Verify all API endpoints work correctly

---

*Generated: 2026-03-09*
*Status: Complete*
