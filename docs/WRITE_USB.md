# Writing the Hyper 32 ISO to USB or DVD

After [building the ISO](BUILD_ISO.md) (`make iso`), use the instructions below
to write `dist/hyper32.iso` to a USB flash drive or burn it to a DVD.

---

## Write to USB flash drive

> ⚠️ **Data loss warning:** `dd` overwrites the target device completely.
> Double-check the device path before running the command.

### Linux

1. Insert your USB drive and find its device path:

   ```bash
   lsblk -d -o NAME,SIZE,MODEL
   # Look for your drive, e.g. /dev/sdb  (14.9G  SanDisk Ultra)
   ```

   Or use:

   ```bash
   sudo dmesg | tail -20   # shows newly attached device
   ```

2. Write the ISO (replace `/dev/sdX` with your actual device — **not** a
   partition like `/dev/sdX1`):

   ```bash
   sudo dd \
       if=dist/hyper32.iso \
       of=/dev/sdX \
       bs=4M \
       status=progress \
       oflag=sync
   ```

3. Eject safely:

   ```bash
   sudo eject /dev/sdX
   ```

### macOS

1. Find your USB device:

   ```bash
   diskutil list
   # Look for your drive, e.g. /dev/disk2
   ```

2. Unmount it (do **not** eject — just unmount):

   ```bash
   diskutil unmountDisk /dev/disk2
   ```

3. Write the ISO (use `/dev/rdisk2` for raw device access — much faster):

   ```bash
   sudo dd \
       if=dist/hyper32.iso \
       of=/dev/rdisk2 \
       bs=4m
   ```

   macOS does not have `status=progress`. Use `Ctrl-T` to send `SIGINFO` and
   check progress, or install `pv` (`brew install pv`) and pipe through it:

   ```bash
   pv dist/hyper32.iso | sudo dd of=/dev/rdisk2 bs=4m
   ```

4. Eject:

   ```bash
   diskutil eject /dev/disk2
   ```

### Windows

Use [Rufus](https://rufus.ie/) (free, open-source):

1. Download and run Rufus.
2. **Device** → select your USB drive.
3. **Boot selection** → click **SELECT** and choose `dist/hyper32.iso`.
4. **Partition scheme** → GPT (for UEFI) or MBR (for BIOS/legacy).
5. Click **START** and accept the warning about data loss.

Alternatively, use [balenaEtcher](https://etcher.balena.io/) for a simpler GUI
on Windows, macOS, or Linux.

---

## Burn to DVD

### Linux — command line (`growisofs`)

```bash
sudo apt-get install -y dvd+rw-tools   # Debian/Ubuntu
sudo growisofs -dvd-compat -Z /dev/sr0=dist/hyper32.iso
```

Replace `/dev/sr0` with your DVD burner device (check `lsblk` or `ls /dev/sr*`).

### Linux / macOS — command line (`cdrecord` / `wodim`)

```bash
# Debian/Ubuntu
sudo apt-get install -y wodim
wodim -v dev=/dev/sr0 driveropts=burnfree,noforcespeed speed=4 \
      -dao dist/hyper32.iso
```

### Windows

Use [ImgBurn](https://www.imgburn.com/) (free):

1. Choose **Write image file to disc**.
2. Select `dist/hyper32.iso` as the source.
3. Click the write button.

---

## Verify the write (recommended)

After writing to USB, verify the data was written correctly.  Get the ISO size
first:

```bash
wc -c < dist/hyper32.iso   # e.g. 15728640 bytes
```

Then read the same number of bytes back from the device and compare checksums:

```bash
ISO_BYTES=$(wc -c < dist/hyper32.iso)
sudo dd if=/dev/sdX bs=512 count=$(( ISO_BYTES / 512 + 1 )) status=none \
    | head -c "${ISO_BYTES}" \
    | sha256sum
sha256sum dist/hyper32.iso
```

Both hashes should match.

---

## Booting

1. Insert the USB / DVD into the target machine.
2. Reboot and enter the **boot menu** (usually F12, F10, Esc, or Del depending
   on the BIOS/UEFI — check your machine's documentation).
3. Select the USB drive or DVD.
4. GRUB loads and presents the **Hyper 32 Live** menu.
5. Select **Hyper 32 Live (x86_64)** (or the verbose-boot variant to see kernel
   messages).
6. The **Hyper 32 Main Menu** TUI appears.

### UEFI Secure Boot

Secure Boot is **not** supported in the current build.  You may need to
**disable Secure Boot** in your UEFI firmware settings before booting.  This is
a known limitation tracked for future work.
