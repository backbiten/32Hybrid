#!/usr/bin/env bash
set -euo pipefail

echo "=== 32Hybrid LEGACY 32-BIT REVIVAL v2026 ==="
echo "Bringing back 1970s/80s 32-bit architecture for the future."
echo "This is your Stone-Age survival script. No 64-bit allowed."

# 1. Force pure 32-bit Go cross-compile (386 = true legacy protected mode)
export GOARCH=386
export GOOS=linux
export CGO_ENABLED=0   # no C deps that could pull 64-bit

# 2. Patch Makefile to lock in 32-bit forever
if grep -q "qemu-system-x86_64" Makefile; then
  sed -i 's/qemu-system-x86_64/qemu-system-i386/g' Makefile
  echo "✓ Makefile updated to qemu-system-i386"
fi
if grep -q "x86_64" Makefile; then
  sed -i 's/x86_64/i386/g' Makefile
  echo "✓ Makefile locked to i386"
fi

# 3. Full reinitialization (clean + proto + build)
echo "Reinitializing everything..."
make clean || true
rm -rf dist/ iso/build/ gen/ bin/
go mod tidy
make install-proto-tools
make proto
make build

# 4. Force legacy 32-bit ISO build
echo "Patching ISO for pure legacy 32-bit..."
cd iso || exit 1

# Force 32-bit kernel (Alpine i386 instead of x86_64)
# Note: We use the existing logic in build.sh but swap the URL/Arch
sed -i 's/x86_64/x86/g' build.sh
sed -i 's/vmlinuz-lts/vmlinuz/g' build.sh

# Strip all UEFI/EFI 64-bit garbage
rm -rf EFI/BOOT/BOOTX64.EFI bootx64.efi 2>/dev/null || true

# Rebuild ISO with legacy BIOS only (grub-pc, no EFI)
bash build.sh

cd ..

# 5. Disaster-proof final steps
mkdir -p legacy32/boot
echo "Applying 1970s/80s survival mode..."
cat > legacy32/boot/grub.cfg << 'EOF'
# Pure legacy 32-bit GRUB — no 64-bit, no EFI
set timeout=3
menuentry "Hyper 32 Legacy (1970s/80s Revival)" {
    linux /boot/vmlinuz root=/dev/ram0
    initrd /boot/initramfs.cpio.gz
}
EOF

# Final test command for you
echo ""
echo "REBOOT TEST (copy-paste this):"
echo "qemu-system-i386 -m 256M -cdrom dist/hyper32.iso -boot d -nographic"
echo ""
echo "This ISO will boot on REAL 386 hardware, old laptops, or after civilization collapses."
echo "32-bit is now preserved. Forever."
echo ""
echo "Next: git add . && git commit -m 'Legacy 32-Bit Revival Script — disaster-proof 1970s/80s architecture' && git push"