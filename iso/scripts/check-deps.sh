#!/usr/bin/env bash
# iso/scripts/check-deps.sh — verify that all ISO build prerequisites are installed
# Exit 0 if everything is present; exit 1 listing what is missing.

set -euo pipefail

REQUIRED_CMDS=(
    xorriso
    grub-mkrescue
    mformat          # from mtools — needed by grub-mkrescue for EFI image
    wget
    cpio
    gzip
    find
)

OPTIONAL_CMDS=(
    qemu-system-x86_64   # for 'make run-iso'
    file
    sha256sum
)

missing=()
for cmd in "${REQUIRED_CMDS[@]}"; do
    if ! command -v "${cmd}" &>/dev/null; then
        missing+=("${cmd}")
    fi
done

if [[ ${#missing[@]} -gt 0 ]]; then
    echo "ERROR: The following required tools are missing:"
    for m in "${missing[@]}"; do
        echo "  - ${m}"
    done
    echo ""
    echo "On Debian/Ubuntu run:"
    echo "  sudo apt-get install -y xorriso grub-pc-bin grub-efi-amd64-bin \\"
    echo "       mtools wget cpio gzip"
    exit 1
fi

warn=()
for cmd in "${OPTIONAL_CMDS[@]}"; do
    if ! command -v "${cmd}" &>/dev/null; then
        warn+=("${cmd}")
    fi
done

if [[ ${#warn[@]} -gt 0 ]]; then
    echo "WARNING: The following optional tools are missing (some targets may not work):"
    for w in "${warn[@]}"; do
        echo "  - ${w}"
    done
fi

echo "All required ISO build dependencies are present."
