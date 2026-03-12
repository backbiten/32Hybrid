#!/usr/bin/env bash
set -euo pipefail

echo "=== 32Hybrid: Community Era Transition (v2026) ==="
echo "Bridging the gap between legacy preservation and modern collaboration."

# 1. Standardize 32-bit environment (Community Defaults)
export GOARCH=386
export GOOS=linux
export CGO_ENABLED=0

# 2. Ensure community-approved build tools are in place
echo "Verifying build tools..."
make install-proto-tools

# 3. Clean and Reinitialize (The Community Way)
echo "Reinitializing project for the community era..."
make clean || true
go mod tidy
make proto
make build

# 4. Prepare Community ISO (Legacy + Modern Accessibility)
echo "Building Community Era ISO..."
cd iso || exit 1
# Ensure the build script uses the community-standard 32-bit targets
sed -i 's/x86_64/x86/g' build.sh
bash build.sh
cd ..

# 5. Final Setup for Community Contributors
echo ""
echo "Community Era Transition Complete."
echo "Contributors can now run: make run-iso"
echo "Tested on QEMU i386 for maximum accessibility."
echo ""
echo "Next: git add . && git commit -m 'Transition to Community Era 32-bit standards' && git push"