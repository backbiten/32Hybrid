#!/usr/bin/env bash
# build-32bit-legacy.sh

echo "Cleaning previous build..."
rm -rf obj libs

echo "Forcing pure 32-bit legacy mode..."
export APP_ABI="armeabi-v7a"   # or edit Application.mk directly

ndk-build -j$(nproc) V=1

# Reinitialize / reinstall steps (example)
# adb uninstall com.yourpackage.legacy
# adb install -r path/to/your-32bit.apk

echo "32-bit legacy reinitialized."