#!/usr/bin/env bash
# iso/build.sh — assemble the Hyper 32 bootable ISO
#
# Usage:
#   ./iso/build.sh              # builds dist/hyper32.iso
#   HYPER32_VERSION=v1.2 ./iso/build.sh
#
# Prerequisites (see docs/BUILD_ISO.md):
#   xorriso, grub-mkrescue, mtools (mformat), wget, cpio, gzip
#   Optional: sha256sum, qemu-system-x86_64

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "${SCRIPT_DIR}")"
ISO_DIR="${SCRIPT_DIR}"
BUILD_DIR="${ISO_DIR}/build"
DIST_DIR="${REPO_ROOT}/dist"
ISO_STAGING="${BUILD_DIR}/iso-staging"

VERSION="${HYPER32_VERSION:-$(git -C "${REPO_ROOT}" describe --tags --always --dirty 2>/dev/null || echo dev)}"
ISO_OUTPUT="${DIST_DIR}/hyper32.iso"

echo "================================================================"
echo "  Hyper 32 ISO build"
echo "  Version : ${VERSION}"
echo "  Output  : ${ISO_OUTPUT}"
echo "================================================================"

# ── 0. Check dependencies ─────────────────────────────────────────────────────
bash "${ISO_DIR}/scripts/check-deps.sh"

# ── 1. Build initramfs ────────────────────────────────────────────────────────
HYPER32_VERSION="${VERSION}" bash "${ISO_DIR}/scripts/build-initramfs.sh"

# ── 2. Download / locate kernel ───────────────────────────────────────────────
# We reuse the host kernel if available (vmlinuz), otherwise download a
# known-good Alpine Linux kernel (statically-built, small, x86_64).
KERNEL_CACHE="${ISO_DIR}/cache/vmlinuz"
ALPINE_KERNEL_URL="https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-standard-3.19.1-x86_64.iso"
ALPINE_KERNEL_SHA256="0e23b35a0af5b5bb39b95e6c9cc0b10d2c285ea9e08d27e0d016e3e4b7bbddc9"

if [[ ! -f "${KERNEL_CACHE}" ]]; then
    # Try host kernel first (fastest, no download).
    # Use a while-read loop so kernel paths with spaces are handled safely.
    while IFS= read -r candidate; do
        [[ -z "${candidate}" ]] && continue
        if [[ -f "${candidate}" ]]; then
            echo "==> Using host kernel: ${candidate}"
            cp "${candidate}" "${KERNEL_CACHE}"
            break
        fi
    done < <(printf '%s\n' /boot/vmlinuz-linux /boot/vmlinuz; \
             ls /boot/vmlinuz-* 2>/dev/null | sort -V | tail -1)
fi

if [[ ! -f "${KERNEL_CACHE}" ]]; then
    echo "==> Downloading Alpine Linux ISO to extract kernel..."
    ALPINE_ISO_CACHE="${ISO_DIR}/cache/alpine-standard.iso"
    if [[ ! -f "${ALPINE_ISO_CACHE}" ]]; then
        wget -q --show-progress -O "${ALPINE_ISO_CACHE}.tmp" "${ALPINE_KERNEL_URL}"
        if command -v sha256sum &>/dev/null; then
            actual=$(sha256sum "${ALPINE_ISO_CACHE}.tmp" | awk '{print $1}')
            if [[ "${actual}" != "${ALPINE_KERNEL_SHA256}" ]]; then
                echo "ERROR: Alpine ISO checksum mismatch!"
                echo "  expected: ${ALPINE_KERNEL_SHA256}"
                echo "  got:      ${actual}"
                rm -f "${ALPINE_ISO_CACHE}.tmp"
                exit 1
            fi
        fi
        mv "${ALPINE_ISO_CACHE}.tmp" "${ALPINE_ISO_CACHE}"
    fi
    # Extract kernel from Alpine ISO
    EXTRACT_TMP="${BUILD_DIR}/alpine-extract"
    mkdir -p "${EXTRACT_TMP}"
    xorriso -osirrox on -indev "${ALPINE_ISO_CACHE}" \
        -extract boot/vmlinuz-lts "${EXTRACT_TMP}/vmlinuz" >/dev/null 2>&1 \
        || xorriso -osirrox on -indev "${ALPINE_ISO_CACHE}" \
            -extract boot/vmlinuz "${EXTRACT_TMP}/vmlinuz" >/dev/null 2>&1
    cp "${EXTRACT_TMP}/vmlinuz" "${KERNEL_CACHE}"
    rm -rf "${EXTRACT_TMP}"
    echo "==> Kernel extracted and cached."
fi

# ── 3. Assemble ISO staging tree ──────────────────────────────────────────────
echo "==> Assembling ISO staging tree..."
rm -rf "${ISO_STAGING}"
mkdir -p \
    "${ISO_STAGING}/boot/grub" \
    "${ISO_STAGING}/EFI/BOOT"

cp "${KERNEL_CACHE}"               "${ISO_STAGING}/boot/vmlinuz"
cp "${BUILD_DIR}/initramfs.cpio.gz" "${ISO_STAGING}/boot/initramfs.cpio.gz"
cp "${BUILD_DIR}/grub.cfg"          "${ISO_STAGING}/boot/grub/grub.cfg"

# ── 4. Build ISO with grub-mkrescue ──────────────────────────────────────────
# grub-mkrescue produces a hybrid ISO that boots on:
#   - Legacy BIOS (via El Torito + grub-pc)
#   - UEFI (via grub-efi embedded EFI image)
# Both modes are handled automatically when grub-efi-amd64-bin and
# grub-pc-bin packages are installed.
echo "==> Running grub-mkrescue..."
mkdir -p "${DIST_DIR}"
grub-mkrescue \
    --output="${ISO_OUTPUT}" \
    --product-name="Hyper 32" \
    --product-version="${VERSION}" \
    -- "${ISO_STAGING}" \
    /boot/grub/grub.cfg="${ISO_STAGING}/boot/grub/grub.cfg"

# ── 5. Post-build info ────────────────────────────────────────────────────────
echo ""
echo "================================================================"
echo "  ISO build complete!"
echo "  Output : ${ISO_OUTPUT}"
echo "  Size   : $(du -sh "${ISO_OUTPUT}" | cut -f1)"
if command -v sha256sum &>/dev/null; then
    echo "  SHA256 : $(sha256sum "${ISO_OUTPUT}" | awk '{print $1}')"
fi
echo ""
echo "  Quick test:"
echo "    make run-iso"
echo ""
echo "  Write to USB:"
echo "    sudo dd if=${ISO_OUTPUT} of=/dev/sdX bs=4M status=progress oflag=sync"
echo "  (replace /dev/sdX with your USB device — see docs/WRITE_USB.md)"
echo "================================================================"
