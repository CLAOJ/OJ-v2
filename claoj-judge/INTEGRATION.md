# Integration Guide: Go Judge with CLAOJ Backend

This guide explains how to integrate the standalone Go judge with your existing CLAOJ backend.

## Overview

The Go judge connects to your existing backend via the bridge protocol on port 9999. Multiple judge instances can connect simultaneously, and the backend distributes submissions among them.

## Prerequisites

1. CLAOJ backend running with bridge enabled (port 9999)
2. Docker and Docker Compose
3. Database access to add judge entries

## Step 1: Verify Backend Bridge

Your backend must have the bridge server enabled. Check your backend configuration:

```go
// In your backend main.go or config
bridgePort := 9999  // Default bridge port
```

The bridge should be listening:

```bash
# Check if bridge is listening
netstat -tlnp | grep 9999

# Or with ss
ss -tlnp | grep 9999
```

## Step 2: Add Judges to Database

Add entries for each judge instance in your `judges` table:

```sql
-- Connect to your database
mysql -u root -p claoj

-- Add judge entries
INSERT INTO judges (name, auth_key, online, last_ip, created_at) VALUES
('Judge-Go-1', 'secure-random-key-1', 0, NULL, NOW()),
('Judge-Go-2', 'secure-random-key-2', 0, NULL, NOW()),
('Judge-Go-3', 'secure-random-key-3', 0, NULL, NOW()),
('Judge-Go-4', 'secure-random-key-4', 0, NULL, NOW())
ON DUPLICATE KEY UPDATE auth_key=VALUES(auth_key);

-- Verify entries
SELECT id, name, online, last_ip FROM judges;
```

**Important:** Use secure random keys in production:

```bash
# Generate secure keys
openssl rand -hex 32
# Or
python3 -c "import secrets; print(secrets.token_hex(32))"
```

## Step 3: Configure Docker Network

The judges need to communicate with the backend over Docker network:

```bash
# Create network (if not exists)
docker network create claoj_judge

# Verify network exists
docker network inspect claoj_judge
```

If your backend is already running, ensure it's on the same network:

```bash
# Connect backend to judge network (if needed)
docker network connect claoj_judge <backend_container_name>
```

## Step 4: Configure Judges

### Option A: Using docker-compose (Recommended)

1. Edit `.env` file with your judge keys:

```bash
JUDGE1_KEY=your-secure-key-1
JUDGE2_KEY=your-secure-key-2
JUDGE3_KEY=your-secure-key-3
JUDGE4_KEY=your-secure-key-4
```

2. Update `docker-compose.yml` if needed:
   - Change `backend` hostname to your backend container name
   - Adjust resource limits

3. Start judges:

```bash
docker-compose up -d
```

### Option B: Manual Docker Run

```bash
# Start judge 1
docker run -d \
  --name judge-go-1 \
  --network claoj_judge \
  -v claoj_problems:/problems \
  -e CLAOJ_JUDGE_NAME=Judge-Go-1 \
  -e CLAOJ_JUDGE_KEY=your-secure-key-1 \
  -p 9998:9998 \
  claoj/judge-go:latest \
  -server backend \
  -port 9999 \
  -name Judge-Go-1 \
  -key your-secure-key-1
```

## Step 5: Verify Connection

Check judge logs:

```bash
docker-compose logs -f judge1
```

Expected output:

```
Connecting to backend:9999 as Judge-Go-1
Handshake successful for judge: Judge-Go-1
Waiting for submissions...
```

Check judge status in database:

```sql
SELECT name, online, last_ip FROM judges;
```

Judges should show `online = 1`.

## Step 6: Test Submission

Submit a test solution through your frontend:

```cpp
// Test submission (C++)
#include <iostream>
int main() {
    std::cout << "Hello, World!" << std::endl;
    return 0;
}
```

Monitor the judge logs:

```bash
docker-compose logs -f judge1 | grep "Starting grading"
```

## Step 7: Migration from Python Judges

### Running Hybrid (Python + Go Judges)

You can run both Python and Go judges simultaneously:

1. Keep existing Python judges running
2. Add Go judges with different names
3. Backend will distribute submissions to all connected judges

### Full Migration

1. Stop Python judges:
```bash
docker stop claoj_judge1 claoj_judge2 claoj_judge3 claoj_judge4
```

2. Start Go judges:
```bash
docker-compose up -d
```

3. Verify all submissions are being processed

## Troubleshooting

### Judge Cannot Connect to Backend

**Symptom:** Connection refused errors in logs

**Solutions:**
1. Verify backend is running: `docker ps | grep backend`
2. Check backend bridge port: `docker logs backend | grep bridge`
3. Test connectivity: `docker run --rm --network claoj_judge alpine nc -zv backend 9999`

### Handshake Failed

**Symptom:** "Authentication failed" or "Judge not found"

**Solutions:**
1. Verify judge name/key in database matches `.env`
2. Check for typos in judge name
3. Ensure judge is not blocked: `SELECT is_blocked FROM judges WHERE name = 'Judge-Go-1'`

### Submissions Not Being Graded

**Symptom:** Judges connected but submissions stay queued

**Solutions:**
1. Check backend logs for submission dispatch
2. Verify problem directories exist: `docker exec -it claoj_judge_go_1 ls /problems`
3. Check judge load: `SELECT name, online FROM judges`

### Permission Denied

**Symptom:** SYS_PTRACE errors

**Solutions:**
1. Ensure `cap_add: - SYS_PTRACE` in docker-compose.yml
2. Check Docker security options: `docker inspect claoj_judge_go_1 | grep SecurityOpt`

## Performance Tuning

### Adjusting Judge Count

Add more judges for higher throughput:

```yaml
# Add to docker-compose.yml
judge5:
  container_name: claoj_judge_go_5
  # ... same as other judges
  command:
    - "-name=Judge-Go-5"
    - "-key=${JUDGE5_KEY}"
```

### Resource Limits

Adjust based on your hardware:

```yaml
deploy:
  resources:
    limits:
      cpus: '2'      # Reduce for lighter problems
      memory: 2048m  # Reduce if memory is not an issue
```

### Compiler Optimization

For faster compilation (less optimized binaries):

```yaml
runtime:
  CPP17:
    flags: "-std=c++17 -O1 -static"  # O1 instead of O2
```

## Monitoring

### Judge Health Check

```bash
# Check all judges
docker-compose ps

# Check individual judge
docker inspect claoj_judge_go_1 --format='{{.State.Health.Status}}'
```

### Metrics to Watch

| Metric | Healthy Value | Alert Threshold |
|--------|---------------|-----------------|
| Judge Count | 4 | < 2 |
| Online Judges | All | Any offline |
| Queue Length | 0 | > 100 |
| Grading Time | < 5s | > 30s |

## Backup and Recovery

### Backup Judge Configuration

```bash
# Save configuration
cp .env .env.backup
cp judge.yml judge.yml.backup
```

### Quick Recovery

```bash
# Restart all judges
docker-compose restart

# Or recreate containers
docker-compose down
docker-compose up -d
```

## Security Considerations

1. **Use secure judge keys** - Generate random 64-character hex strings
2. **Isolate judge network** - Use separate Docker network
3. **Limit capabilities** - Only add required capabilities
4. **Run as non-root** - Dockerfile uses `judge` user
5. **Enable TLS** - Set `secure: true` for production

## Next Steps

1. Set up monitoring and alerting
2. Configure log aggregation
3. Implement automated scaling
4. Set up staging environment for testing

---

For more information, see:
- `README.md` - Judge usage and configuration
- `judge.example.yml` - All configuration options
- `docker-compose.yml` - Docker deployment configuration
