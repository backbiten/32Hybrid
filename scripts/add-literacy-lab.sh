#!/usr/bin/env bash
set -euo pipefail

echo "=== 32Hybrid LITERACY LAB EXPANSION (Ethical Offline Edition) ==="
echo "Adds safe, air-gapped computer literacy tools & menu for community use"
echo "1990s/2000s Pentium-era feel — no internet, no external tools, pure education"
echo "For Temple of Set neighborhood literacy initiatives only"

# 1. Ensure pure 32-bit context
export GOARCH=386
export GOOS=linux
export CGO_ENABLED=0

# 2. Reinitialize base build if needed (calls your existing reinit if present)
if [ -f scripts/reinit-32hybrid.sh ]; then
  echo "Running existing reinitialization..."
  ./scripts/reinit-32hybrid.sh
elif [ -f scripts/pure-32bit-legacy.sh ]; then
  echo "Running pure 32-bit legacy setup..."
  ./scripts/pure-32bit-legacy.sh
else
  echo "Running Makefile reinit / clean build..."
  make clean || true
  make reinit || make all || echo "No reinit/all target — continuing anyway"
fi

# 3. Create Literacy Lab directory structure (offline content goes here)
LIT_DIR="legacy32/literacy-lab"
mkdir -p "$LIT_DIR"/{tutorials,basics,retro,simulations}

echo "Creating offline literacy content placeholders..."

# Basic welcome / ethical lock file
cat > "$LIT_DIR/README.txt" << 'EOF'
TEMPLE OF SET COMMUNITY LITERACY LAB
Offline 32-bit Education Module – 1970s to 2000s Revival
For middle/lower/poor neighborhoods only – Air-gapped by design
Goal: Help people go from computer-illiterate to comfortable with basics

No internet. No corporate/government tracking. Pure learning.
EOF

# Example: simple text tutorial files (expand these later!)
cat > "$LIT_DIR/basics/what-is-a-computer.txt" << 'EOF'
What is a Computer? (1990s Style Explanation)

Back in the 1990s, a computer was a box with:
- CPU (the brain – like a fast Pentium chip)
- RAM (short-term memory)
- Hard drive (long-term storage)
- Keyboard, mouse, monitor

Turn it on → DOS prompt or Windows 95 appears.
Type commands or click icons to do work/play.

Try it: At the boot prompt, type 'help' and press Enter.
EOF

cat > "$LIT_DIR/tutorials/command-line-basics.txt" << 'EOF'
Command-Line Basics (like old DOS or early Linux terminals)

cd ..          → go up one folder
ls             → list files (or 'dir' in DOS)
cat file.txt   → read a text file
echo "Hello"   → print text to screen

Practice safely: These only affect this isolated system.
EOF

cat > "$LIT_DIR/retro/1990s-pentium-feel.txt" << 'EOF'
1990s Pentium Era Nostalgia

- 90 MHz to 200 MHz CPUs
- Windows 95/98 boot sound
- Dial-up modem sounds (we simulate offline)
- Games like Math Blaster, Oregon Trail
- Word processing in WordPerfect style (big buttons, no distractions)

Goal: Feel how normal people first learned computers
EOF