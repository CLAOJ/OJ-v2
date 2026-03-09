# Setup script for CLAOJ Go Judge

set -e

echo "========================================"
echo "  CLAOJ Go Judge Setup"
echo "========================================"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}Note: Some commands require root privileges${NC}"
fi

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

echo ""
echo "Step 1: Checking prerequisites..."
echo "----------------------------------------"

# Check Docker
if command_exists docker; then
    echo -e "${GREEN}✓ Docker installed${NC}"
else
    echo -e "${RED}✗ Docker not found${NC}"
    echo "Please install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check Docker Compose
if command_exists docker-compose || docker compose version >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Docker Compose installed${NC}"
else
    echo -e "${RED}✗ Docker Compose not found${NC}"
    echo "Please install Docker Compose"
    exit 1
fi

# Check Go (for local development)
if command_exists go; then
    echo -e "${GREEN}✓ Go installed ($(go version))${NC}"
else
    echo -e "${YELLOW}! Go not found (required only for local development)${NC}"
fi

echo ""
echo "Step 2: Creating Docker network and volumes..."
echo "----------------------------------------"

# Create judge network if not exists
if ! docker network inspect claoj_judge >/dev/null 2>&1; then
    docker network create claoj_judge
    echo -e "${GREEN}✓ Created network: claoj_judge${NC}"
else
    echo -e "${GREEN}✓ Network already exists: claoj_judge${NC}"
fi

# Create problems volume if not exists
if ! docker volume inspect claoj_problems >/dev/null 2>&1; then
    docker volume create claoj_problems
    echo -e "${GREEN}✓ Created volume: claoj_problems${NC}"
else
    echo -e "${GREEN}✓ Volume already exists: claoj_problems${NC}"
fi

echo ""
echo "Step 3: Creating configuration..."
echo "----------------------------------------"

# Create logs directory
mkdir -p logs/judge1 logs/judge2 logs/judge3 logs/judge4
echo -e "${GREEN}✓ Created logs directories${NC}"

# Create .env file if not exists
if [ ! -f .env ]; then
    cat > .env << 'EOF'
# Judge authentication keys
# Change these to secure random strings!
JUDGE1_KEY=judge1-key-CHANGE-ME
JUDGE2_KEY=judge2-key-CHANGE-ME
JUDGE3_KEY=judge3-key-CHANGE-ME
JUDGE4_KEY=judge4-key-CHANGE-ME
EOF
    echo -e "${GREEN}✓ Created .env file${NC}"
    echo -e "${YELLOW}! IMPORTANT: Edit .env and change the judge keys!${NC}"
else
    echo -e "${GREEN}✓ .env file already exists${NC}"
fi

# Create judge config if not exists
if [ ! -f judge.yml ]; then
    cp judge.example.yml judge.yml
    echo -e "${GREEN}✓ Created judge.yml${NC}"
else
    echo -e "${GREEN}✓ judge.yml already exists${NC}"
fi

echo ""
echo "Step 4: Building judge image..."
echo "----------------------------------------"

docker build -t claoj/judge-go:latest .

echo -e "${GREEN}✓ Judge image built${NC}"

echo ""
echo "Step 5: Database setup..."
echo "----------------------------------------"

echo -e "${YELLOW}Note: You need to add judge entries to your backend database${NC}"
echo ""
echo "Run this SQL in your backend database:"
echo ""
cat << 'SQL'
-- Add judge entries (change the keys to match your .env file!)
INSERT INTO judges (name, auth_key, online, last_ip, created_at) VALUES
('Judge-Go-1', 'judge1-key-CHANGE-ME', 0, NULL, NOW()),
('Judge-Go-2', 'judge2-key-CHANGE-ME', 0, NULL, NOW()),
('Judge-Go-3', 'judge3-key-CHANGE-ME', 0, NULL, NOW()),
('Judge-Go-4', 'judge4-key-CHANGE-ME', 0, NULL, NOW())
ON DUPLICATE KEY UPDATE auth_key=VALUES(auth_key);
SQL

echo ""
read -p "Press Enter after you've run the SQL commands..."

echo ""
echo "Step 6: Starting judges..."
echo "----------------------------------------"

docker-compose up -d

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Setup Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Useful commands:"
echo "  docker-compose logs -f     - View judge logs"
echo "  docker-compose ps          - Check judge status"
echo "  docker-compose down        - Stop all judges"
echo "  docker-compose restart     - Restart all judges"
echo ""
echo "Individual judge control:"
echo "  docker start/stop/restart claoj_judge_go_1"
echo ""
