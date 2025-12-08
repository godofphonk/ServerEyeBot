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

# Check if .env files exist
if [ ! -f .env.dev ] && [ ! -f .env.prod ]; then
    echo -e "${RED}❌ .env.dev or .env.prod not found. Please configure them first.${NC}"
    echo -e "${YELLOW}Copy .env.example to .env.dev and .env.prod and fill in your values.${NC}"
    exit 1
fi

# Validate .env files
echo -e "${GREEN}Validating environment configuration...${NC}"
if [ -f .env.dev ] && grep -q "your_telegram_bot_token_here" .env.dev; then
    echo -e "${RED}❌ Please update TELEGRAM_TOKEN in .env.dev${NC}"
    exit 1
fi

if [ -f .env.prod ] && grep -q "your_production_telegram_bot_token_here" .env.prod; then
    echo -e "${RED}❌ Please update TELEGRAM_TOKEN in .env.prod${NC}"
    exit 1
fi

echo -e "\n${GREEN}Setup complete!${NC}"
echo -e "\n${BLUE}Next steps:${NC}"
echo "1. Update .env.dev and .env.prod with your actual values"
echo "2. Generate GHCR auth config: ./scripts/generate-ghcr-auth.sh <username> <token>"
echo "3. Start dev bot: docker-compose -f deployments/docker-compose.dev.yml up -d"
echo "4. Start Watchtower: docker-compose -f deployments/docker-compose.watchtower.yml up -d"
echo ""
echo -e "${YELLOW}For production deployment:${NC}"
echo "1. Update .env.prod with production values"
echo "2. Start prod bot: docker-compose -f deployments/docker-compose.prod.yml up -d"
