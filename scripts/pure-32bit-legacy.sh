#!/usr/bin/env bash
set -euo pipefail

echo "=== Switching 32Hybrid to pure 32-bit legacy mode ==="

# Optional: set environment for 32-bit Go cross-compile
export GOARCH=386
export CGO_ENABLED=1   # if you have C dependencies

# Patch Makefile or config if needed (example — customize!)
if grep -q "GOARCH=amd64" Makefile; then
  sed -i 's/GOARCH=amd64/GOARCH=386/g' Makefile
  echo "Makefile patched to GOARCH=386"
fi

# Re-run full reinitialization in 32-bit context
if [ -f scripts/reinit-32hybrid.sh ]; then
  ./scripts/reinit-32hybrid.sh
else
  echo "Warning: scripts/reinit-32hybrid.sh not found. Skipping full re-init."
fi

# Extra legacy steps (e.g. force BIOS boot, strip 64-bit files from ISO)
echo "Applying pure legacy tweaks..."
# Example: remove any 64-bit EFI stubs from iso/ if they exist
rm -f iso/bootx64.efi iso/EFI/BOOT/BOOTX64.EFI 2>/dev/null || true

echo "Now building pure 32-bit legacy."
echo " - Use qemu-system-i386 (not qemu-system-x86_64) for testing"
echo " - ISO should boot in legacy BIOS/CSM mode only"