#!/bin/bash

# ServerEye Bot Environment Validation Script
# Validates required environment variables are set

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Required environment variables
REQUIRED_VARS=(
    "TELEGRAM_TOKEN"
    "REDIS_ADDRESS"
    "DATABASE_URL"
    "ENVIRONMENT"
)

# Optional but recommended variables
OPTIONAL_VARS=(
    "REDIS_PASSWORD"
    "KAFKA_ENABLED"
    "KAFKA_BROKERS"
    "JWT_SECRET"
    "LOG_LEVEL"
)

# Environment-specific requirements
PROD_REQUIRED_VARS=(
    "JWT_SECRET"
    "KAFKA_ENABLED"
)

# Function to check if variable is set and not empty
check_var() {
    local var_name=$1
    local var_value=${!var_name}
    
    if [ -z "$var_value" ]; then
        echo -e "${RED}✗ Missing required variable: $var_name${NC}"
        return 1
    else
        echo -e "${GREEN}✓ $var_name is set${NC}"
        return 0
    fi
}

# Function to check optional variable
check_optional_var() {
    local var_name=$1
    local var_value=${!var_name}
    
    if [ -z "$var_value" ]; then
        echo -e "${YELLOW}⚠ Optional variable not set: $var_name${NC}"
    else
        echo -e "${GREEN}✓ $var_name is set${NC}"
    fi
}

# Function to validate environment value
validate_environment() {
    local env=$1
    case $env in
        dev|development)
            echo -e "${GREEN}✓ Environment: development${NC}"
            ;;
        staging)
            echo -e "${GREEN}✓ Environment: staging${NC}"
            ;;
        prod|production)
            echo -e "${GREEN}✓ Environment: production${NC}"
            # Check production-specific requirements
            for var in "${PROD_REQUIRED_VARS[@]}"; do
                check_var "$var" || exit 1
            done
            ;;
        *)
            echo -e "${RED}✗ Invalid environment: $env (must be dev, staging, or prod)${NC}"
            exit 1
            ;;
    esac
}

# Function to validate Kafka configuration if enabled
validate_kafka() {
    if [ "${KAFKA_ENABLED:-false}" = "true" ]; then
        echo -e "\n${GREEN}Validating Kafka configuration...${NC}"
        check_var "KAFKA_BROKERS" || exit 1
        
        # Validate broker format
        if [[ ! "$KAFKA_BROKERS" =~ ^[a-zA-Z0-9.-]+:[0-9]+(,[a-zA-Z0-9.-]+:[0-9]+)*$ ]]; then
            echo -e "${RED}✗ Invalid KAFKA_BROKERS format. Expected: host1:port1,host2:port2${NC}"
            exit 1
        fi
        
        echo -e "${GREEN}✓ Kafka configuration is valid${NC}"
    fi
}

# Function to validate database URL
validate_database_url() {
    local db_url=$1
    if [[ ! "$db_url" =~ ^postgres:// ]]; then
        echo -e "${RED}✗ Invalid DATABASE_URL format. Expected: postgres://user:password@host:port/dbname${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Database URL format is valid${NC}"
}

# Main validation
main() {
    echo -e "${GREEN}ServerEye Bot Environment Validation${NC}"
    echo -e "${GREEN}===================================${NC}\n"
    
    # Check required variables
    echo -e "${GREEN}Checking required variables...${NC}"
    missing_required=0
    for var in "${REQUIRED_VARS[@]}"; do
        if ! check_var "$var"; then
            missing_required=$((missing_required + 1))
        fi
    done
    
    if [ $missing_required -gt 0 ]; then
        echo -e "\n${RED}✗ Validation failed: $missing_required required variables are missing${NC}"
        exit 1
    fi
    
    # Check optional variables
    echo -e "\n${GREEN}Checking optional variables...${NC}"
    for var in "${OPTIONAL_VARS[@]}"; do
        check_optional_var "$var"
    done
    
    # Validate environment
    echo -e "\n${GREEN}Validating environment...${NC}"
    validate_environment "$ENVIRONMENT"
    
    # Validate specific configurations
    echo -e "\n${GREEN}Validating specific configurations...${NC}"
    validate_database_url "$DATABASE_URL"
    validate_kafka
    
    # Check for weak secrets in production
    if [ "$ENVIRONMENT" = "prod" ] || [ "$ENVIRONMENT" = "production" ]; then
        if [ "${JWT_SECRET}" = "change-me-in-production" ] || [ "${JWT_SECRET}" = "dev-secret-change-in-production" ]; then
            echo -e "\n${RED}✗ Production environment detected with default JWT_SECRET! Please set a strong secret.${NC}"
            exit 1
        fi
    fi
    
    echo -e "\n${GREEN}✅ All validations passed! Environment is properly configured.${NC}"
}

# Run validation
main "$@"
