#!/usr/bin/env bash
# rotate-jwt-secret.sh - JWT Secret Rotation Helper Script
#
# This script assists with zero-downtime JWT secret rotation by generating
# a new secret and providing instructions for updating environment variables.
#
# Usage:
#   ./scripts/rotate-jwt-secret.sh
#
# The script will:
# 1. Generate a new 32-character base64 secret
# 2. Display current and new secrets
# 3. Provide step-by-step instructions for rotation
#
# For automated deployments, you can set these variables in your CI/CD:
# - JWT_SECRET_CURRENT: New secret
# - JWT_SECRET_PREVIOUS: Current secret (from previous deployment)
#
# Security Note:
# - Never commit secrets to version control
# - Store secrets in a secure secret management system (e.g., AWS Secrets Manager, HashiCorp Vault)
# - Rotate secrets monthly or after any suspected compromise

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Generate new secret
NEW_SECRET=$(openssl rand -base64 32)

echo -e "${BLUE}============================================================================${NC}"
echo -e "${BLUE}JWT Secret Rotation Helper${NC}"
echo -e "${BLUE}============================================================================${NC}"
echo ""

# Check if JWT_SECRET_CURRENT or JWT_SECRET is set
CURRENT_SECRET=""
if [ -n "${JWT_SECRET_CURRENT:-}" ]; then
    CURRENT_SECRET="$JWT_SECRET_CURRENT"
    echo -e "${GREEN}✓ Found JWT_SECRET_CURRENT in environment${NC}"
elif [ -n "${JWT_SECRET:-}" ]; then
    CURRENT_SECRET="$JWT_SECRET"
    echo -e "${YELLOW}⚠ Found legacy JWT_SECRET in environment${NC}"
    echo -e "${YELLOW}  Consider migrating to JWT_SECRET_CURRENT for rotation support${NC}"
else
    echo -e "${RED}✗ No JWT secret found in environment${NC}"
    echo -e "${YELLOW}  This appears to be an initial setup${NC}"
fi

echo ""
echo -e "${BLUE}Generated New Secret:${NC}"
echo -e "${GREEN}$NEW_SECRET${NC}"
echo ""

if [ -n "$CURRENT_SECRET" ]; then
    echo -e "${BLUE}Current Secret (first 8 chars):${NC}"
    echo -e "${YELLOW}${CURRENT_SECRET:0:8}...${NC}"
    echo ""
fi

echo -e "${BLUE}============================================================================${NC}"
echo -e "${BLUE}Rotation Instructions${NC}"
echo -e "${BLUE}============================================================================${NC}"
echo ""

if [ -n "$CURRENT_SECRET" ]; then
    echo -e "${YELLOW}Step 1: Update Environment Variables${NC}"
    echo ""
    echo "Set the following environment variables in your deployment environment:"
    echo ""
    echo -e "  ${GREEN}JWT_SECRET_CURRENT=${NC}$NEW_SECRET"
    echo -e "  ${GREEN}JWT_SECRET_PREVIOUS=${NC}<paste your FULL current secret here>"
    echo ""
    echo -e "  ${YELLOW}⚠ IMPORTANT: Use the FULL, untruncated current secret (shown above as ${CURRENT_SECRET:0:8}...)${NC}"
    echo -e "  ${YELLOW}  Do NOT copy just the preview. The secret must be complete for validation to work.${NC}"
    echo ""
    echo -e "${YELLOW}Step 2: Deploy to All Instances${NC}"
    echo ""
    echo "Deploy the updated configuration to all API server instances."
    echo "During this phase:"
    echo "  • New tokens will be signed with JWT_SECRET_CURRENT"
    echo "  • Old tokens will still validate using JWT_SECRET_PREVIOUS"
    echo "  • Zero downtime for users"
    echo ""
    echo -e "${YELLOW}Step 3: Wait for Token Expiration${NC}"
    echo ""
    echo "Wait for the maximum token lifetime to pass:"
    echo "  • Access tokens: 15 minutes"
    echo "  • Refresh tokens: 7 days"
    echo ""
    echo "Recommended: Wait 7 days to ensure all refresh tokens have expired."
    echo ""
    echo -e "${YELLOW}Step 4: Remove Previous Secret${NC}"
    echo ""
    echo "After the rotation window, remove JWT_SECRET_PREVIOUS:"
    echo ""
    echo -e "  ${GREEN}JWT_SECRET_CURRENT=${NC}$NEW_SECRET"
    echo -e "  ${RED}# JWT_SECRET_PREVIOUS= ${YELLOW}(remove or leave empty)${NC}"
    echo ""
else
    echo -e "${YELLOW}Initial Setup${NC}"
    echo ""
    echo "Set the JWT secret in your environment:"
    echo ""
    echo -e "  ${GREEN}JWT_SECRET_CURRENT=${NC}$NEW_SECRET"
    echo ""
    echo "OR (for backward compatibility):"
    echo ""
    echo -e "  ${GREEN}JWT_SECRET=${NC}$NEW_SECRET"
    echo ""
fi

echo -e "${BLUE}============================================================================${NC}"
echo -e "${BLUE}Security Best Practices${NC}"
echo -e "${BLUE}============================================================================${NC}"
echo ""
echo "• Rotate secrets monthly or after any suspected compromise"
echo "• Store secrets in a secure secret management system"
echo "• Never commit secrets to version control"
echo "• Use different secrets for development, staging, and production"
echo "• Monitor for unusual authentication patterns during rotation"
echo "• Keep audit logs of secret rotation events"
echo ""
echo -e "${GREEN}✓ Rotation preparation complete${NC}"
echo ""
