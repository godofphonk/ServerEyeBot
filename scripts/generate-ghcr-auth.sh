#!/bin/bash

# ServerEye Bot GHCR Authentication Config Generator
# Generates config.json for Watchtower to authenticate with GitHub Container Registry

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to show usage
show_usage() {
    echo "Usage: $0 <github_username> <github_token>"
    echo ""
    echo "Generates config.json for Watchtower GHCR authentication"
    echo ""
    echo "Arguments:"
    echo "  github_username  Your GitHub username"
    echo "  github_token     Your GitHub Personal Access Token with 'read:packages' scope"
    echo ""
    echo "Example:"
    echo "  $0 myusername ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    echo ""
    echo "Output will be saved to: deployments/config.json"
}

# Check if arguments are provided
if [ $# -ne 2 ]; then
    echo -e "${RED}Error: Missing required arguments${NC}"
    echo ""
    show_usage
    exit 1
fi

GITHUB_USERNAME=$1
GITHUB_TOKEN=$2

# Validate inputs
if [ -z "$GITHUB_USERNAME" ]; then
    echo -e "${RED}Error: GitHub username cannot be empty${NC}"
    exit 1
fi

if [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${RED}Error: GitHub token cannot be empty${NC}"
    exit 1
fi

# Check if token has the right format (starts with ghp_)
if [[ ! "$GITHUB_TOKEN" =~ ^ghp_ ]]; then
    echo -e "${YELLOW}Warning: GitHub token should start with 'ghp_'${NC}"
    echo -e "${YELLOW}Make sure your token has 'read:packages' scope${NC}"
    echo ""
fi

# Generate base64 encoded auth string
AUTH_STRING="${GITHUB_USERNAME}:${GITHUB_TOKEN}"
BASE64_AUTH=$(echo -n "$AUTH_STRING" | base64)

# Create config.json
CONFIG_FILE="deployments/config.json"
cat > "$CONFIG_FILE" <<EOF
{
  "auths": {
    "ghcr.io": {
      "auth": "${BASE64_AUTH}"
    }
  }
}
EOF

echo -e "${GREEN}✅ Generated GHCR authentication config${NC}"
echo -e "${GREEN}📁 Saved to: ${CONFIG_FILE}${NC}"
echo ""
echo -e "${YELLOW}⚠️  Important notes:${NC}"
echo "1. Ensure your GitHub token has 'read:packages' scope"
echo "2. Keep ${CONFIG_FILE} secure and never commit it to version control"
echo "3. The config will be used by Watchtower to pull images from GHCR"
echo ""
echo -e "${GREEN}🚀 You can now start Watchtower:${NC}"
echo "  docker-compose -f deployments/docker-compose.watchtower.yml up -d"
