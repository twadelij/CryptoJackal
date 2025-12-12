#!/bin/bash
# =============================================================================
# CryptoJackal Proxmox VM Setup Script
# =============================================================================
# This script creates a dedicated LXC container for CryptoJackal testing
# =============================================================================

set -e

# Configuration
PROXMOX_HOST="192.168.2.125"
PROXMOX_PORT="8006"
PROXMOX_USER="root@pam"
PROXMOX_PASS="Handd03kV00rdeel"
NODE="proxmox"
STORAGE="local-lvm"
TEMPLATE_STORAGE="DS_ProxMox"

# New container settings
VMID="100"
HOSTNAME="cryptojackal-test"
MEMORY="4096"
SWAP="512"
CORES="2"
DISK="20"
BRIDGE="vmbr0"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║         CryptoJackal Proxmox Setup Script                    ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"

# Get authentication ticket
echo -e "\n${YELLOW}🔐 Authenticating with Proxmox...${NC}"
AUTH_RESPONSE=$(curl -k -s -d "username=$PROXMOX_USER&password=$PROXMOX_PASS" \
    "https://$PROXMOX_HOST:$PROXMOX_PORT/api2/json/access/ticket")

TICKET=$(echo "$AUTH_RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['ticket'])")
CSRF=$(echo "$AUTH_RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['CSRFPreventionToken'])")

if [ -z "$TICKET" ]; then
    echo -e "${RED}❌ Authentication failed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Authenticated successfully${NC}"

# Function to make API calls
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    if [ "$method" = "GET" ]; then
        curl -k -s -b "PVEAuthCookie=$TICKET" \
            "https://$PROXMOX_HOST:$PROXMOX_PORT$endpoint"
    elif [ "$method" = "POST" ]; then
        curl -k -s -b "PVEAuthCookie=$TICKET" \
            -H "CSRFPreventionToken: $CSRF" \
            -X POST -d "$data" \
            "https://$PROXMOX_HOST:$PROXMOX_PORT$endpoint"
    elif [ "$method" = "DELETE" ]; then
        curl -k -s -b "PVEAuthCookie=$TICKET" \
            -H "CSRFPreventionToken: $CSRF" \
            -X DELETE \
            "https://$PROXMOX_HOST:$PROXMOX_PORT$endpoint"
    fi
}

# List existing containers
echo -e "\n${YELLOW}📋 Listing existing containers...${NC}"
CONTAINERS=$(api_call GET "/api2/json/nodes/$NODE/lxc" | python3 -c "
import sys,json
data = json.load(sys.stdin)['data']
for c in data:
    print(f\"{c['vmid']}:{c['name']}:{c['status']}\")
")

if [ -n "$CONTAINERS" ]; then
    echo -e "${CYAN}Existing containers:${NC}"
    echo "$CONTAINERS" | while IFS=: read vmid name status; do
        echo "  - $vmid: $name ($status)"
    done
    
    echo -e "\n${RED}⚠️  WARNING: This will stop and remove ALL existing containers!${NC}"
    read -p "Do you want to proceed? (yes/no): " confirm
    
    if [ "$confirm" != "yes" ]; then
        echo -e "${YELLOW}Aborted by user${NC}"
        exit 0
    fi
    
    # Stop and remove containers
    echo -e "\n${YELLOW}🗑️  Removing existing containers...${NC}"
    echo "$CONTAINERS" | while IFS=: read vmid name status; do
        echo "  Stopping $vmid ($name)..."
        api_call POST "/api2/json/nodes/$NODE/lxc/$vmid/status/stop" "" >/dev/null 2>&1 || true
        sleep 2
        echo "  Removing $vmid..."
        api_call DELETE "/api2/json/nodes/$NODE/lxc/$vmid" >/dev/null 2>&1 || true
        sleep 1
    done
    echo -e "${GREEN}✅ Containers removed${NC}"
fi

# Check for Ubuntu template
echo -e "\n${YELLOW}📦 Checking for Ubuntu template...${NC}"
TEMPLATES=$(api_call GET "/api2/json/nodes/$NODE/storage/$TEMPLATE_STORAGE/content" | python3 -c "
import sys,json
data = json.load(sys.stdin).get('data', [])
for t in data:
    if 'vztmpl' in t.get('content', '') and 'ubuntu' in t.get('volid', '').lower():
        print(t['volid'])
        break
" 2>/dev/null || echo "")

if [ -z "$TEMPLATES" ]; then
    echo -e "${YELLOW}No Ubuntu template found. Downloading...${NC}"
    api_call POST "/api2/json/nodes/$NODE/aplinfo" \
        "storage=$TEMPLATE_STORAGE&template=ubuntu-24.04-standard_24.04-2_amd64.tar.zst"
    sleep 30
    TEMPLATES="$TEMPLATE_STORAGE:vztmpl/ubuntu-24.04-standard_24.04-2_amd64.tar.zst"
fi

echo -e "${GREEN}✅ Using template: $TEMPLATES${NC}"

# Create new container
echo -e "\n${YELLOW}🚀 Creating CryptoJackal test container...${NC}"
CREATE_RESULT=$(api_call POST "/api2/json/nodes/$NODE/lxc" \
    "vmid=$VMID&hostname=$HOSTNAME&ostemplate=$TEMPLATES&storage=$STORAGE&rootfs=$STORAGE:$DISK&memory=$MEMORY&swap=$SWAP&cores=$CORES&net0=name=eth0,bridge=$BRIDGE,ip=dhcp&unprivileged=1&features=nesting=1&start=1&password=CryptoJackal123")

echo "$CREATE_RESULT" | python3 -m json.tool 2>/dev/null || echo "$CREATE_RESULT"

echo -e "\n${GREEN}✅ Container created!${NC}"
echo -e "${CYAN}Container Details:${NC}"
echo "  - VMID: $VMID"
echo "  - Hostname: $HOSTNAME"
echo "  - Memory: ${MEMORY}MB"
echo "  - Cores: $CORES"
echo "  - Disk: ${DISK}GB"
echo "  - Password: CryptoJackal123"

# Wait for container to start
echo -e "\n${YELLOW}⏳ Waiting for container to start...${NC}"
sleep 10

# Get container IP
echo -e "\n${YELLOW}🌐 Getting container IP address...${NC}"
for i in {1..30}; do
    IP=$(api_call GET "/api2/json/nodes/$NODE/lxc/$VMID/interfaces" 2>/dev/null | python3 -c "
import sys,json
try:
    data = json.load(sys.stdin).get('data', [])
    for iface in data:
        if iface.get('name') == 'eth0':
            for addr in iface.get('inet', '').split():
                if addr and not addr.startswith('127.'):
                    print(addr.split('/')[0])
                    break
except: pass
" 2>/dev/null || echo "")
    
    if [ -n "$IP" ]; then
        break
    fi
    sleep 2
done

if [ -n "$IP" ]; then
    echo -e "${GREEN}✅ Container IP: $IP${NC}"
else
    echo -e "${YELLOW}⚠️  Could not determine IP. Check Proxmox console.${NC}"
fi

echo -e "\n${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                    Setup Complete!                           ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo -e "\n${GREEN}Next steps:${NC}"
echo "1. SSH into container: ssh root@$IP (password: CryptoJackal123)"
echo "2. Install dependencies: apt update && apt install -y docker.io docker-compose nodejs npm git"
echo "3. Clone CryptoJackal: git clone https://github.com/twadelij/CryptoJackal.git"
echo "4. Run setup: cd CryptoJackal && ./setup.sh"
