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

# Errors go to stderr so that they are not captured as the value of whichever
# function was being evaluated when they occurred.
print_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# api <method> <url> [json-body]
#
# Writes the response body to stdout. Returns non-zero, having explained why, if
# the request could not be made or the API answered with anything other than a
# 2xx. curl is deliberately not given -f, which would discard the response body
# that explains the failure.
api() {
    local method=$1
    local url=$2
    local data=${3:-}
    local -a curl_args=(
      -sS
      -X "$method"
      -H "Authorization: Bearer $NETLIFY_AUTH_TOKEN"
      -H "Content-Type: application/json"
      -w $'\n%{http_code}'
    )
    if [[ -n "$data" ]]; then
        curl_args+=(-d "$data")
    fi
    local response
    if ! response=$(curl "${curl_args[@]}" "$url"); then
        print_error "$method $url: could not reach the Netlify API"
        return 1
    fi
    # -w appended the status code on its own trailing line.
    local status=${response##*$'\n'}
    local body=${response%$'\n'*}
    if [[ "$status" != 2* ]]; then
        print_error "$method $url: Netlify API returned HTTP $status"
        print_error "Response body: $body"
        return 1
    fi
    printf '%s' "$body"
}

get_current_prod_branch() {
    local site_json
    site_json=$(api GET "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID") || return 1
    local current_branch
    current_branch=$(jq -r '.build_settings.repo_branch // empty' <<< "$site_json")
    if [[ -z "$current_branch" ]]; then
        print_error "Site $NETLIFY_SITE_ID reports no production branch. Netlify answers"
        print_error "with a reduced projection of a public site's settings, omitting"
        print_error "build_settings, when the caller is not authorized to read them, so"
        print_error "check that NETLIFY_AUTH_TOKEN is current and has write access."
        return 1
    fi
    echo "$current_branch"
}

# Writes the site's allowed branches to stdout as a JSON array, or nothing at
# all if the site has no such setting. It is left to the caller to preserve that
# distinction: replacing an absent setting with a list of this script's own
# making would silently narrow which branches Netlify will build.
get_current_allowed_branches() {
    local site_json
    site_json=$(api GET "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID") || return 1
    jq -c '.build_settings.allowed_branches // empty' <<< "$site_json"
}

promote_branch() {
    local old_branch=$1
    local new_branch=$2
    print_status "Promoting branch $new_branch to production"
    local allowed_branches
    allowed_branches=$(get_current_allowed_branches) || return 1
    local payload
    if [[ -z "$allowed_branches" ]]; then
        print_warning "Site $NETLIFY_SITE_ID has no allowed branches setting; leaving it"
        print_warning "alone. Confirm that $old_branch is configured for branch deploys."
        payload=$(jq -nc --arg branch "$new_branch" \
          '{build_settings: {repo_branch: $branch}}')
    else
        allowed_branches=$(jq -c --arg b "$old_branch" \
          'if index($b) then . else . + [$b] end' <<< "$allowed_branches")
        payload=$(jq -nc --arg branch "$new_branch" --argjson allowed "$allowed_branches" \
          '{build_settings: {repo_branch: $branch, allowed_branches: $allowed}}')
    fi
    api PUT "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID" "$payload" > /dev/null || return 1
    # A write the API accepts is not proof that the setting changed, so read it
    # back before claiming the branch was promoted.
    local actual_branch
    actual_branch=$(get_current_prod_branch) || return 1
    if [[ "$actual_branch" != "$new_branch" ]]; then
        print_error "Netlify accepted the update, but the production branch of site"
        print_error "$NETLIFY_SITE_ID reads back as \"$actual_branch\" instead of"
        print_error "\"$new_branch\". The site has not been promoted."
        return 1
    fi
    print_success "Branch $new_branch promoted to production"
}

trigger_prod_build() {
    print_status "Triggering production build + deployment"
    local deploy_response
    deploy_response=$(api POST "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID/builds") || return 1
    local deploy_id
    deploy_id=$(jq -r '.id // empty' <<< "$deploy_response") || return 1
    if [[ -z "$deploy_id" ]]; then
        print_error "Netlify accepted the production build request but returned no"
        print_error "deployment ID. Response body: $deploy_response"
        return 1
    fi
    print_success "Production build + deployment triggered. Deployment ID: $deploy_id"
}

trigger_branch_build() {
    local branch=$1
    print_status "Triggering branch $branch build + deployment"
    local deploy_response
    deploy_response=$(api POST "https://api.netlify.com/api/v1/sites/$NETLIFY_SITE_ID/builds?branch=$branch") || return 1
    local deploy_id
    deploy_id=$(jq -r '.id // empty' <<< "$deploy_response") || return 1
    if [[ -z "$deploy_id" ]]; then
        print_error "Netlify accepted the build request for $branch but returned no"
        print_error "deployment ID. Response body: $deploy_response"
        return 1
    fi
    print_success "Build + deployment triggered for $branch. Deployment ID: $deploy_id"
}

# Netlify derives a branch deploy's subdomain by lowercasing the branch name and
# replacing every character that is not alphanumeric.
branch_subdomain() {
    printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g'
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
    echo "✅ Branch build + deployment triggered for: $old_branch"
    echo
    echo "Production site: https://$DOMAIN_NAME"
    echo
    echo "The old documentation will remain available at: https://$(branch_subdomain "$old_branch").$DOMAIN_NAME"
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
    current_prod_branch=$(get_current_prod_branch) || exit 1
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

    promote_branch "$old_prod_branch" "$new_prod_branch" || exit 1
    trigger_prod_build || exit 1

    trigger_branch_build "$old_prod_branch" || exit 1

    display_summary "$new_prod_branch" "$old_prod_branch"

    print_success "Documentation promotion completed!"
}

main "$@"
