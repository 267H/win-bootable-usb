# Windows 11 Bootable USB Creator for Mac M1

A simple Go script to automate creating a bootable Windows 11 USB drive on macOS (Apple Silicon).

## What & Why

- **What it does**
    1. Prompts for a Windows 11 ISO file on your machine.
    2. Lists and validates the target USB device.
    3. Formats the USB as FAT32 with a GPT partition scheme.
    4. Mounts the ISO and copies all files except `install.wim`.
    5. Splits `install.wim` into multiple `.swm` files so it fits on a FAT32 volume.
    6. Unmounts/ejects both the ISO and USB.

- **Why use it**
    - Automates common manual steps (`diskutil`, `hdiutil`, `rsync`, `wimlib-imagex`)
    - Ensures proper splitting of the large Windows install image for FAT32
    - Reduces mistakes—fewer copy-&-paste shell commands

## Prerequisites

- macOS running on Apple Silicon (M1, M2, etc.)
- [Homebrew](https://brew.sh/) installed
- Install required tools:
  ```bash
  brew install wimlib
  ```

## Usage

1. **Build the binary**
   ```bash
   go build -o win11-usb
   ```

2. **Run the script**
   ```bash
   ./win11-usb
   ```

3. **Follow prompts**
    - Enter the full path to your Windows 11 ISO (e.g., `~/Downloads/Win11.iso`)
    - Choose the USB device identifier (e.g., `/dev/disk2`)
    - Confirm formatting (all data on the USB will be erased)

4. **Boot**
    - After “Success!”, eject and insert the USB into your target PC
    - In BIOS/UEFI, select your USB in UEFI mode and install Windows 11

## Notes

- The script assumes the formatted USB volume will be named `WINUSB`.
- If your USB device name differs, adjust the `usbName` variable in the code.
- Always back up any important data before formatting a drive.
