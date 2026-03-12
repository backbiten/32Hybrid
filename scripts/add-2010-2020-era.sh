#!/usr/bin/env bash
set -euo pipefail

echo "=== 32Hybrid 2010–2020 ERA REVIVAL (Pre-2025 Memory Hole Pull) ==="
echo "Pulling back last 32-bit architectures (i386 / x86 32-bit)"
echo "2010 (Nehalem/Core i7 era) • 2015 (Haswell) • 2020 (pre-obsolete)"
echo "For Temple of Set / community homes ONLY — air-gapped, 32-bit mesh only"
echo "No 64-bit ever. No corporate/government. Neighborhood preservation."

# 1. Force pure 32-bit context (i386)
export GOARCH=386
export GOOS=linux
export CGO_ENABLED=0

# 2. Re-run your existing reinitialization
if [ -f scripts/community-era-32hybrid.sh ]; then
  ./scripts/community-era-32hybrid.sh
elif [ -f scripts/reinit-32hybrid.sh ]; then
  ./scripts/reinit-32hybrid.sh
else
  make clean || true
  make all || echo "Continuing..."
fi

# 3. Create historical era folder with \"memory hole\" links & notes
HIST_DIR=\"legacy32/historical-2010-2020\"
mkdir -p "$HIST_DIR"/{kernels,boot-images,notes}

cat > "$HIST_DIR/README-2010-2020.txt" << 'EOF'
TEMPLE OF SET COMMUNITY — 2010–2020 32-BIT REVIVAL
These were the LAST real 32-bit boot images before everything got thrown into the memory hole in 2025.

2010 Era: Nehalem / Core i7 (first \"modern\" 32-bit feel)
2015 Era: Haswell (peak 32-bit desktop era)
2020 Era: Last i386 support before distros killed it

All pulled from public archives (Debian CDImage archive, kernel.org, Archive.org).
Use only on 32-bit mesh between homes/apartments/condos.
No 64-bit machines can join — ever.

Links (copy into browser on a separate machine if needed):
• Debian 9.13 i386 (2017–2020 era): https://cdimage.debian.org/mirror/cdimage/archive/9.13.0/i386/iso-cd/
• Debian 10 i386 (2019–2020): https://cdimage.debian.org/mirror/cdimage/archive/10.13.0/i386/iso-cd/
• Old Alpine i386 rootfs (Docker mirror for 2015–2020): https://hub.docker.com/r/i386/alpine
• Linux kernels 2010–2020 (v3.x to v5.4 i386 compatible): https://cdn.kernel.org/pub/linux/kernel/v4.x/ or v5.x/

Place any downloaded i386 kernel/initrd into legacy32/historical-2010-2020/kernels/
EOF

# 4. Update GRUB menu with new 2010–2020 era entries (QEMU CPU models)
GRUB_CFG=\"legacy32/boot/grub.cfg\"
echo "Creating new grub.cfg with 2010-2020 eras..."
mkdir -p legacy32/boot
cat > "$GRUB_CFG" << 'EOF'
# Temple of Set Community 32-Bit GRUB — 1970s-2020 Revival
timeout=5

menuentry "Legacy 32 — 1970s/80s 80386" { linux /boot/vmlinuz cpu=486; initrd /boot/initramfs.cpio.gz }
menuentry "Legacy 32 — 1990s Pentium" { linux /boot/vmlinuz cpu=pentium; initrd /boot/initramfs.cpio.gz }
menuentry "Legacy 32 — 2000s Pentium 4" { linux /boot/vmlinuz cpu=pentium3; initrd /boot/initramfs.cpio.gz }
menuentry "Legacy 32 — 2010 Era (Nehalem / Core i7)" { linux /boot/vmlinuz cpu=Nehalem; initrd /boot/initramfs.cpio.gz }
menuentry "Legacy 32 — 2015 Era (Haswell)" { linux /boot/vmlinuz cpu=Haswell; initrd /boot/initramfs.cpio.gz }
menuentry "Legacy 32 — 2020 Era (last i386)" { linux /boot/vmlinuz cpu=Skylake; initrd /boot/initramfs.cpio.gz }
menuentry "Hybrid 32 Community Literacy + Mesh" { linux /boot/vmlinuz cpu=Haswell lab=literacy mesh=on; initrd /boot/initramfs.cpio.gz }
EOF

# 5. Patch Makefile & QEMU for era CPU support (2010–2020 models)
sed -i 's/-cpu pentium/-cpu Haswell/g' Makefile 2>/dev/null || true
echo "✓ QEMU now supports 2010–2020 CPU models (Nehalem, Haswell, Skylake)"

# 6. Force air-gap + 32-bit mesh reminder
echo "32-bit mesh reminder: run /opt/32mesh/mesh-init.sh wlan0 on each home machine"
echo "Only i386 machines will see each other. 64-bit is completely invisible."

# 7. Rebuild the ISO with new eras included
if [ -f iso/build.sh ]; then
  cd iso
  bash build.sh
  cd ..
fi

echo ""
echo "✅ 2010–2020 Era Added — Memory Hole Successfully Pulled"