#!/usr/bin/env bash
# iso/scripts/build-initramfs.sh — build a minimal BusyBox-based initramfs
#
# Produces: iso/build/initramfs.cpio.gz
#
# The script downloads a statically-compiled BusyBox binary (no internet
# required after the first build thanks to a local cache), then assembles
# a minimal rootfs and packs it as a compressed CPIO archive.
#
# Requires: wget, cpio, gzip, find, chmod, mkdir, ln
# Optional: sha256sum (for checksum verification)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISO_DIR="$(dirname "${SCRIPT_DIR}")"
REPO_ROOT="$(dirname "${ISO_DIR}")"
BUILD_DIR="${ISO_DIR}/build"
ROOTFS_DIR="${BUILD_DIR}/rootfs"
BUSYBOX_VERSION="${BUSYBOX_VERSION:-1.36.1}"
BUSYBOX_URL="https://busybox.net/downloads/binaries/${BUSYBOX_VERSION}-x86_64-linux-musl/busybox"
BUSYBOX_SHA256="b8cc24c9574d809e7279c3be349795c5d5ceb6fdf19ca709f80cde50e47de314"
BUSYBOX_CACHE="${ISO_DIR}/cache/busybox-${BUSYBOX_VERSION}"
VERSION="${HYPER32_VERSION:-$(git -C "${REPO_ROOT}" describe --tags --always --dirty 2>/dev/null || echo dev)}"

echo "==> Building Hyper 32 initramfs (version: ${VERSION})"

# ── Prepare directories ───────────────────────────────────────────────────────
rm -rf "${ROOTFS_DIR}"
mkdir -p \
    "${ROOTFS_DIR}/bin" \
    "${ROOTFS_DIR}/sbin" \
    "${ROOTFS_DIR}/usr/bin" \
    "${ROOTFS_DIR}/usr/sbin" \
    "${ROOTFS_DIR}/usr/local/bin" \
    "${ROOTFS_DIR}/etc" \
    "${ROOTFS_DIR}/proc" \
    "${ROOTFS_DIR}/sys" \
    "${ROOTFS_DIR}/dev" \
    "${ROOTFS_DIR}/tmp" \
    "${ROOTFS_DIR}/mnt" \
    "${ROOTFS_DIR}/root"

mkdir -p "${ISO_DIR}/cache"

# ── BusyBox ───────────────────────────────────────────────────────────────────
if [[ ! -f "${BUSYBOX_CACHE}" ]]; then
    echo "==> Downloading BusyBox ${BUSYBOX_VERSION}..."
    wget -q --show-progress -O "${BUSYBOX_CACHE}.tmp" "${BUSYBOX_URL}"

    if command -v sha256sum &>/dev/null; then
        actual=$(sha256sum "${BUSYBOX_CACHE}.tmp" | awk '{print $1}')
        if [[ "${actual}" != "${BUSYBOX_SHA256}" ]]; then
            echo "ERROR: BusyBox checksum mismatch!"
            echo "  expected: ${BUSYBOX_SHA256}"
            echo "  got:      ${actual}"
            rm -f "${BUSYBOX_CACHE}.tmp"
            exit 1
        fi
        echo "==> Checksum verified."
    else
        echo "WARNING: sha256sum not found; skipping checksum verification."
    fi

    mv "${BUSYBOX_CACHE}.tmp" "${BUSYBOX_CACHE}"
fi

cp "${BUSYBOX_CACHE}" "${ROOTFS_DIR}/bin/busybox"
chmod +x "${ROOTFS_DIR}/bin/busybox"

# Create BusyBox symlinks
echo "==> Installing BusyBox applets..."
for applet in sh ash bash cat cp dd df echo env find grep gzip \
               head hostname ifconfig ip kill less ls mkdir mknod \
               mount mv poweroff ps reboot rm rmdir sed sleep \
               sort tail touch uname umount vi wget; do
    ln -sf /bin/busybox "${ROOTFS_DIR}/bin/${applet}" 2>/dev/null || true
done
ln -sf /bin/busybox "${ROOTFS_DIR}/sbin/mdev"
ln -sf /bin/busybox "${ROOTFS_DIR}/sbin/init"

# ── Copy static rootfs overlays ───────────────────────────────────────────────
echo "==> Copying rootfs overlays..."
cp "${ISO_DIR}/rootfs/init"                        "${ROOTFS_DIR}/init"
cp "${ISO_DIR}/rootfs/etc/motd"                    "${ROOTFS_DIR}/etc/motd"
cp "${ISO_DIR}/rootfs/usr/local/bin/hyper32-menu"  "${ROOTFS_DIR}/usr/local/bin/hyper32-menu"
chmod +x "${ROOTFS_DIR}/init" "${ROOTFS_DIR}/usr/local/bin/hyper32-menu"

# Write version file
echo "${VERSION}" > "${ROOTFS_DIR}/etc/hyper32-version"

# ── Hyper 32 Go binaries (optional — included if already built) ───────────────
for bin in controlplane runner avdclient; do
    src="${REPO_ROOT}/bin/${bin}"
    if [[ -f "${src}" ]]; then
        echo "==> Bundling Hyper 32 binary: ${bin}"
        cp "${src}" "${ROOTFS_DIR}/usr/local/bin/${bin}"
        chmod +x "${ROOTFS_DIR}/usr/local/bin/${bin}"
    fi
done
# Convenience symlink: hyper32 -> controlplane
if [[ -f "${ROOTFS_DIR}/usr/local/bin/controlplane" ]]; then
    ln -sf /usr/local/bin/controlplane "${ROOTFS_DIR}/usr/local/bin/hyper32"
fi

# Patch GRUB config with actual version string
sed "s/__VERSION__/${VERSION}/g" \
    "${ISO_DIR}/grub/grub.cfg" > "${BUILD_DIR}/grub.cfg"

# ── Pack initramfs ────────────────────────────────────────────────────────────
echo "==> Packing initramfs..."
(
    cd "${ROOTFS_DIR}"
    find . | cpio -oH newc | gzip -9 > "${BUILD_DIR}/initramfs.cpio.gz"
)
echo "==> initramfs written to ${BUILD_DIR}/initramfs.cpio.gz"
echo "    size: $(du -sh "${BUILD_DIR}/initramfs.cpio.gz" | cut -f1)"
