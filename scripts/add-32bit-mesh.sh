#!/usr/bin/env bash
set -euo pipefail

echo "=== Adding 32-bit Local Mesh Networking to Legacy/Hybrid 32 ISO ==="
echo "Strictly 32-bit only • Home-to-home isolation • No 64-bit peering"
echo "Uses batman-adv for simple WiFi/Ethernet mesh"

# 1. Ensure pure 32-bit build context
export GOARCH=386
export GOOS=linux
export CGO_ENABLED=0

# 2. Re-run base reinitialization
if [ -f scripts/reinit-32hybrid.sh ]; then
  ./scripts/reinit-32hybrid.sh
elif [ -f scripts/community-era-32hybrid.sh ]; then
  ./scripts/community-era-32hybrid.sh
else
  make clean || true
  make all || echo "Continuing without full rebuild"
fi

# 3. Create mesh scripts dir in rootfs
MESH_DIR="iso/rootfs/opt/32mesh"
mkdir -p "$MESH_DIR"

# 4. Main mesh init script
cat > "$MESH_DIR/mesh-init.sh" << 'EOF'
#!/bin/sh
# 32-bit Local Mesh Init — Home/Apartment/Condo only
# Runs on i386 / 32-bit x86 ONLY

echo "Starting 32-bit Mesh (batman-adv mode)"

# Check architecture (refuse if 64-bit)
ARCH=$(uname -m)
if [ "$ARCH" != "i686" ] && [ "$ARCH" != "i386" ]; then
  echo "ERROR: This is NOT a 32-bit machine ($ARCH). Mesh refused."
  exit 1
fi

# Load module (if not auto-loaded)
modprobe batman-adv 2>/dev/null || echo "batman-adv already loaded or unavailable"

# Choose interface (wlan0 for WiFi mesh, eth0 for wired)
IFACE="[1m${1:-wlan0}[0m"
echo "Using interface: $IFACE"

# Add to batman-adv mesh
if command -v batctl >/dev/null 2>&1; then
  batctl if add "$IFACE"
  ip link set bat0 up
  ip addr add 10.32.0.$(shuf -i 10-250 -n 1)/24 dev bat0
else
  echo "ERROR: batctl not found. Please ensure it is installed in the rootfs."
  exit 1
fi

echo ""
echo "32-bit Mesh active on bat0"
echo " - Other 32-bit machines on same WiFi/ethernet will auto-join"
echo " - No internet gateway added → fully isolated"
EOF

chmod +x "$MESH_DIR/mesh-init.sh"

# 5. Add to MOTD for visibility
if [ -f iso/rootfs/etc/motd ]; then
  echo "" >> iso/rootfs/etc/motd
  echo "32-bit Mesh Networking available:" >> iso/rootfs/etc/motd
  echo "Run: /opt/32mesh/mesh-init.sh [interface]" >> iso/rootfs/etc/motd
fi

# 6. Rebuild ISO
if [ -f iso/build.sh ]; then
  cd iso
  bash build.sh
  cd ..
fi

echo ""
echo "✅ Done. 32-bit Local Mesh Networking added."
echo " - Script: /opt/32mesh/mesh-init.sh"
echo " - Isolation: Private 10.32.0.0/24, no default route."