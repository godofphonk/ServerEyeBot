#!/bin/bash

# ServerEye Bot Server Setup Script
# Sets up the server environment for running ServerEyeBot with Watchtower

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}ServerEye Bot Server Setup${NC}"
echo -e "${BLUE}============================${NC}\n"

# Create necessary directories
echo -e "${GREEN}Creating directories...${NC}"
sudo mkdir -p /var/log/servereye
sudo mkdir -p /etc/servereye/secrets
sudo chown -R $USER:$USER /var/log/servereye
sudo chown -R $USER:$USER /etc/servereye

# Create Docker network if not exists
echo -e "${GREEN}Setting up Docker network...${NC}"
if ! docker network ls | grep -q servereye-network; then
    docker network create servereye-network
    echo -e "${GREEN}✅ Created servereye-network${NC}"
else
    echo -e "${YELLOW}⚠️  servereye-network already exists${NC}"
fi

# Check if .env.local exists
if [ ! -f .env.local ]; then
    echo -e "${RED}❌ .env.local not found. Please configure it first.${NC}"
    echo -e "${YELLOW}Copy .env.example to .env.local and fill in your values.${NC}"
    exit 1
fi

# Validate .env.local
echo -e "${GREEN}Validating environment configuration...${NC}"
if grep -q "your_telegram_bot_token_here" .env.local; then
    echo -e "${RED}❌ Please update TELEGRAM_TOKEN in .env.local${NC}"
    exit 1
fi

echo -e "\n${GREEN}Setup complete!${NC}"
echo -e "\n${BLUE}Next steps:${NC}"
echo "1. Update .env.local with your actual values"
echo "2. Generate GHCR auth config: ./scripts/generate-ghcr-auth.sh <username> <token>"
echo "3. Start dev bot: docker-compose -f deployments/docker-compose.dev.yml up -d"
echo "4. Start Watchtower: docker-compose -f deployments/docker-compose.watchtower.yml up -d"
echo ""
echo -e "${YELLOW}For production deployment:${NC}"
echo "1. Update .env.prod with production values"
echo "2. Start prod bot: docker-compose -f deployments/docker-compose.prod.yml up -d"
