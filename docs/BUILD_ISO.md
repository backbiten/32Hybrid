# Building the Hyper 32 Live ISO

This document explains how to build the **Hyper 32 live ISO** — a bootable disc
image you can write to a USB flash drive or burn to a DVD to revive a machine
using Hyper 32.

---

## Prerequisites

### Required tools

| Tool | Package (Debian/Ubuntu) | Purpose |
|------|------------------------|---------|
| `xorriso` | `xorriso` | ISO creation / extraction |
| `grub-mkrescue` | `grub-common` | Bootloader assembly |
| `grub-pc-bin` | `grub-pc-bin` | BIOS boot support |
| `grub-efi-amd64-bin` | `grub-efi-amd64-bin` | UEFI boot support |
| `mformat` | `mtools` | EFI partition creation inside grub-mkrescue |
| `wget` | `wget` | Download BusyBox / Alpine kernel |
| `cpio` | `cpio` | Initramfs packing |
| `gzip` | `gzip` | Initramfs compression |

Install all at once on **Debian / Ubuntu**:

```bash
sudo apt-get update
sudo apt-get install -y \
    xorriso grub-common grub-pc-bin grub-efi-amd64-bin \
    mtools wget cpio gzip
```

On **Arch Linux**:

```bash
sudo pacman -S xorriso grub mtools wget cpio gzip
```

On **Fedora / RHEL**:

```bash
sudo dnf install -y xorriso grub2-tools grub2-efi-x64 mtools wget cpio gzip
```

### Optional tools

| Tool | Purpose |
|------|---------|
| `qemu-system-x86_64` | Boot the ISO locally for testing (`make run-iso`) |
| `sha256sum` | Integrity verification of downloaded artefacts |
| `file` | Inspect ISO type |

Install QEMU on Debian/Ubuntu:

```bash
sudo apt-get install -y qemu-system-x86
```

---

## Build the ISO

### 1. Build Hyper 32 binaries and the ISO in one step

```bash
make iso
```

This will:
1. Compile the Hyper 32 Go binaries into `bin/` (`make build`).
2. Download a statically-compiled BusyBox binary (cached in `iso/cache/`).
3. Locate or download a bootable Linux kernel (host `/boot/vmlinuz*` is used
   if available, otherwise a minimal Alpine Linux kernel is downloaded).
4. Assemble a minimal BusyBox-based initramfs containing the Hyper 32 binaries
   and the `hyper32-menu` TUI script.
5. Run `grub-mkrescue` to produce `dist/hyper32.iso`.

### 2. Output

```
dist/hyper32.iso   — the bootable hybrid ISO (BIOS + UEFI)
```

The build prints the file size and SHA-256 checksum on completion.

---

## Test in QEMU

```bash
make run-iso
```

This boots `dist/hyper32.iso` in QEMU with 512 MB RAM, sending the console to
your terminal (`-nographic -serial mon:stdio`).

You should see:
1. GRUB menu with two entries — **Hyper 32 Live (x86_64)** and a verbose-boot
   variant.
2. The boot banner (ASCII art + version).
3. The **Hyper 32 Main Menu** TUI (pure shell, no external TUI libraries).

To exit QEMU: press `Ctrl-A` then `x`.

### QEMU with a graphical window

```bash
qemu-system-x86_64 -m 512M -cdrom dist/hyper32.iso -boot d -no-reboot
```

### QEMU with UEFI (requires OVMF)

```bash
sudo apt-get install -y ovmf
qemu-system-x86_64 \
    -m 512M \
    -bios /usr/share/ovmf/OVMF.fd \
    -cdrom dist/hyper32.iso \
    -boot d \
    -no-reboot
```

---

## Customising the build

### Embed a specific version string

```bash
HYPER32_VERSION=v0.2.0 make iso
```

### Skip the Go build (re-use existing binaries)

```bash
make -s build          # normal Go build
HYPER32_VERSION=dev bash iso/build.sh
```

### Use a custom kernel

Copy your kernel to `iso/cache/vmlinuz` before running `make iso`:

```bash
cp /path/to/custom-vmlinuz iso/cache/vmlinuz
make iso
```

---

## Directory layout

```
iso/
├── build.sh                    # Top-level ISO assembly script
├── grub/
│   └── grub.cfg                # GRUB bootloader configuration
├── rootfs/
│   ├── init                    # PID-1 init script
│   ├── etc/
│   │   └── motd                # Boot banner
│   └── usr/local/bin/
│       └── hyper32-menu        # TUI shell menu
├── scripts/
│   ├── build-initramfs.sh      # Builds the cpio.gz initramfs
│   └── check-deps.sh           # Prerequisite checker
└── cache/                      # Downloaded artefacts (git-ignored)
```

---

## Current status vs future work

| Feature | Status |
|---------|--------|
| BIOS boot via GRUB | ✅ Working |
| UEFI boot via GRUB (hybrid ISO) | ✅ Working (requires grub-efi-amd64-bin) |
| BusyBox-based initramfs | ✅ Working |
| Hyper 32 TUI menu | ✅ Working (pure shell) |
| Hyper 32 binaries bundled into ISO | ✅ Working (built then embedded) |
| Persistent storage overlay | 🔲 Future work |
| Graphical UI (beyond TUI) | 🔲 Future work |
| ARM64 / i386 support | 🔲 Future work |
| Network boot (iPXE) | 🔲 Future work |
| Automated CI ISO build + artifact upload | 🔲 Future work |
