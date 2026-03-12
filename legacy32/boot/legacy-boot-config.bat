@echo off
:: UEFI fallback and legacy compatibility configuration
bcdedit /set {default} path \EFI\BOOT\bootx86.efi
bcdedit /set removememory 0x10000000
echo Legacy boot configuration applied. Please reboot into legacy/CSM/hybrid mode manually via BIOS.