#!/bin/bash

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

NETLIFY_SITE_ID="71b4c2e1-5e8b-4927-ad1f-b475bae59e90" 
NETLIFY_AUTH_TOKEN="${NETLIFY_AUTH_TOKEN:-}"
DOMAIN_NAME="docs.kargo.io"
DOCS_DIR="docs"
REMOTE="upstream"

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

get_current_prod_branch() {
    local current_branch=$(curl -sS \
      -H "Authorization: Bearer $NETLIFY_AUTH_TOKEN" \
      -H "Content-Type: application/json" \
      "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID" | \
      jq -r '.build_settings.repo_branch // "main"')
    echo "$current_branch"
}

get_current_allowed_branches() {
    allowed_branches=$(curl -sS \
      -H "Authorization: Bearer $NETLIFY_AUTH_TOKEN" \
      -H "Content-Type: application/json" \
      "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID" | \
      jq -r '.build_settings.allowed_branches // []')
    echo "$allowed_branches"
}

promote_branch() {
    local old_branch=$1
    local new_branch=$2
    print_status "Promoting branch $new_branch to production"
    local allowed_branches=$(get_current_allowed_branches)
    if [[ ! "$allowed_branches" =~ "$old_branch" ]]; then
        allowed_branches=$(echo "$allowed_branches" | jq ". + [\"$old_prod_branch\"]")
    fi
    curl -sS -X PUT \
      -H "Authorization: Bearer $NETLIFY_AUTH_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"build_settings\": {\"branch\": \"$new_branch\", \"allowed_branches\": $allowed_branches}}" \
      "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID" > /dev/null
    print_success "Branch $new_branch promoted to production"
}

trigger_prod_build() {
    print_status "Triggering production build + deployment"
    local deploy_response=$(curl -sS -X POST \
      -H "Authorization: Bearer $NETLIFY_AUTH_TOKEN" \
      -H "Content-Type: application/json" \
      "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID/builds")
    local deploy_id=$(echo "$deploy_response" | jq -r '.id')
    print_success "Production build + deployment triggered. Deployment ID: $deploy_id"
}

trigger_branch_build() {
    local branch=$1
    print_status "Triggering branch $branch build + deployment"
    local deploy_response
    deploy_response=$(curl -sS -X POST \
      -H "Authorization: Bearer $NETLIFY_AUTH_TOKEN" \
      -H "Content-Type: application/json" \
      "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID/builds?branch=$branch")
    local deploy_id
    deploy_id=$(echo "$deploy_response" | jq -r '.id')
    print_success "Build + deployment triggered for $branch. Deployment ID: $deploy_id"
}

# Function to display summary
display_summary() {
    local new_branch=$1
    local old_branch=$2
    echo
    echo "=================================================================="
    echo -e "${GREEN}Documentation Promotion Summary${NC}"
    echo "=================================================================="
    echo "✅ Production site updated to branch: $new_branch"
    echo "✅ Production build + deployment triggered from: $new_branch"
    echo "✅ Branch deploy (slot) created for: $old_branch"
    echo "✅ Branch build + deployment triggered for: $old_branch"
    echo
    echo "Production site: https://$DOMAIN_NAME"
    echo
    echo "The old documentation will remain available at: https://$old_branch.$DOMAIN_NAME"
    echo
    echo "Use the Netlify Dashboard to follow up on build + deployment progress: https://app.netlify.com/sites/$NETLIFY_SITE_ID"
    echo "=================================================================="
}

main() {
    local new_prod_branch="${1:-}"

    echo "=================================================================="
    echo -e "${BLUE}Kargo Documentation Promotion Script${NC}"
    echo "=================================================================="

    if [[ -z "$new_prod_branch" ]]; then
        print_error "Usage: $0 <new-prod-branch>"
        print_error "Example: $0 release-1.4"
        exit 1
    fi

    if [[ -z "$NETLIFY_AUTH_TOKEN" ]]; then
        print_error "NETLIFY_AUTH_TOKEN environment variable is not set. Please set it and try again."
        exit 1
    fi

    local current_prod_branch
    current_prod_branch=$(get_current_prod_branch)
    print_status "Current production branch: $current_prod_branch"
    print_status "New production branch: $new_prod_branch"

    if [[ "$current_prod_branch" == "$new_prod_branch" ]]; then
        print_warning "Branch ($new_prod_branch) is already the production branch. Nothing to do."
        exit 0
    fi

    echo
    read -p "Do you want to proceed with promoting branch $new_prod_branch to production? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Operation cancelled"
        exit 0
    fi

    local old_prod_branch
    old_prod_branch="$current_prod_branch"

    echo
    print_status "Starting documentation promotion process..."

    promote_branch "$old_prod_branch" "$new_prod_branch"
    trigger_prod_build

    trigger_branch_build "$old_prod_branch"

    display_summary "$new_prod_branch" "$old_prod_branch"

    print_success "Documentation promotion completed!"
}

main "$@"
