#!/bin/bash

# CCDash Nginx Setup Script - Minimal Version
# This script sets up minimal nginx proxy configuration for CCDash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NGINX_CONF_NAME="ccdash.conf"
NGINX_TEMPLATE_NAME="ccdash.conf.template"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
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

detect_nginx_config_dir() {
    if [ -d "/opt/homebrew/etc/nginx/servers" ]; then
        NGINX_SERVERS_DIR="/opt/homebrew/etc/nginx/servers"
        NGINX_CONF_DIR="/opt/homebrew/etc/nginx"
        return 0
    elif [ -d "/usr/local/etc/nginx/servers" ]; then
        NGINX_SERVERS_DIR="/usr/local/etc/nginx/servers"
        NGINX_CONF_DIR="/usr/local/etc/nginx"
        return 0
    elif [ -d "/etc/nginx/sites-available" ]; then
        NGINX_SITES_AVAILABLE="/etc/nginx/sites-available"
        NGINX_SITES_ENABLED="/etc/nginx/sites-enabled"
        return 0
    else
        print_error "Could not detect nginx configuration directory"
        return 1
    fi
}

setup_config() {
    print_info "Setting up nginx configuration..."
    
    # Check if personal config exists
    if [ ! -f "$SCRIPT_DIR/$NGINX_CONF_NAME" ]; then
        if [ -f "$SCRIPT_DIR/$NGINX_TEMPLATE_NAME" ]; then
            print_info "Creating personal config from template..."
            cp "$SCRIPT_DIR/$NGINX_TEMPLATE_NAME" "$SCRIPT_DIR/$NGINX_CONF_NAME"
            print_success "Created $NGINX_CONF_NAME from template"
            print_info "You can now customize $NGINX_CONF_NAME as needed (it's gitignored)"
        else
            print_error "Template file $NGINX_TEMPLATE_NAME not found"
            return 1
        fi
    else
        print_success "Personal config $NGINX_CONF_NAME already exists"
    fi
}

show_manual_steps() {
    print_info "Manual setup steps:"
    echo
    
    if ! detect_nginx_config_dir; then
        print_error "Please install nginx first"
        return 1
    fi
    
    if [ -n "$NGINX_SERVERS_DIR" ]; then
        echo "1. Copy configuration:"
        echo "   sudo cp $SCRIPT_DIR/$NGINX_CONF_NAME $NGINX_SERVERS_DIR/"
        echo
        echo "2. Update main nginx config to avoid port conflicts:"
        echo "   sudo sed -i '' 's/listen.*8080/listen       8888/' $NGINX_CONF_DIR/nginx.conf"
        echo
    else
        echo "1. Copy configuration:"
        echo "   sudo cp $SCRIPT_DIR/$NGINX_CONF_NAME $NGINX_SITES_AVAILABLE/"
        echo "   sudo ln -sf $NGINX_SITES_AVAILABLE/$NGINX_CONF_NAME $NGINX_SITES_ENABLED/"
        echo "   sudo rm -f $NGINX_SITES_ENABLED/default"
        echo
    fi
    
    echo "3. Test and reload nginx:"
    echo "   sudo nginx -t"
    echo "   sudo nginx -s reload"
    echo
    echo "4. Start CCDash services:"
    echo "   cd ../backend && go run cmd/server/main.go  # In one terminal"
    echo "   cd ../frontend && npm run dev              # In another terminal"
    echo
    echo "5. Access CCDash:"
    echo "   http://localhost"
    echo
}

check_services() {
    print_info "Checking services status..."
    
    # Check backend
    if pgrep -f "go run.*server/main.go\|ccdash-server" > /dev/null; then
        print_success "Backend is running (port 6060)"
    else
        print_warning "Backend is not running"
        echo "  Start with: cd ../backend && go run cmd/server/main.go"
    fi
    
    # Check frontend
    if pgrep -f "next-server\|npm run dev" > /dev/null; then
        print_success "Frontend is running (port 3000)"
    else
        print_warning "Frontend is not running"
        echo "  Start with: cd ../frontend && npm run dev"
    fi
    
    # Check nginx
    if pgrep nginx > /dev/null; then
        print_success "Nginx is running"
    else
        print_warning "Nginx is not running"
        echo "  Start with: sudo nginx"
    fi
}

main() {
    echo "CCDash Nginx Setup - Template Configuration"
    echo "==========================================="
    echo
    
    setup_config
    echo
    show_manual_steps
    check_services
    
    echo
    print_info "Configuration Summary:"
    echo "  - Frontend: http://localhost -> http://127.0.0.1:3000"
    echo "  - API: http://localhost/api -> http://127.0.0.1:6060/api"
    echo "  - Template: $SCRIPT_DIR/$NGINX_TEMPLATE_NAME (tracked in git)"
    echo "  - Personal config: $SCRIPT_DIR/$NGINX_CONF_NAME (gitignored)"
}

main "$@"