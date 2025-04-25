package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// runCommand executes a shell command and prints output
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getUSBDevice prompts for and validates the USB device identifier
func getUSBDevice() (string, error) {
	fmt.Println("Listing available disks:")
	if err := runCommand("diskutil", "list"); err != nil {
		return "", fmt.Errorf("failed to list disks: %v", err)
	}
	fmt.Print("Enter the USB device identifier (e.g., /dev/disk2): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	device := strings.TrimSpace(scanner.Text())
	if !strings.HasPrefix(device, "/dev/disk") {
		return "", fmt.Errorf("invalid device identifier: %s", device)
	}
	return device, nil
}

// getISOPath prompts for and validates the Windows 11 ISO path
func getISOPath() (string, error) {
	fmt.Print("Enter the full path to the Windows 11 ISO (e.g., ~/Downloads/Win11.iso): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	isoPath := strings.TrimSpace(scanner.Text())
	isoPath = strings.ReplaceAll(isoPath, "~", os.Getenv("HOME"))
	if !strings.HasSuffix(isoPath, ".iso") {
		return "", fmt.Errorf("file is not an ISO: %s", isoPath)
	}
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		return "", fmt.Errorf("ISO file does not exist: %s", isoPath)
	}
	return isoPath, nil
}

// unmountUSB unmounts all partitions of the USB device
func unmountUSB(device string) error {
	fmt.Printf("Unmounting all partitions on %s...\n", device)
	if err := runCommand("diskutil", "unmountDisk", device); err != nil {
		return fmt.Errorf("failed to unmount USB: %v", err)
	}
	return nil
}

// formatUSB formats the USB as FAT32 with GPT
func formatUSB(device, usbName string) error {
	// Unmount the USB first
	if err := unmountUSB(device); err != nil {
		return err
	}
	fmt.Printf("Formatting %s as FAT32 with GPT...\n", device)
	if err := runCommand("diskutil", "eraseDisk", "FAT32", usbName, "GPT", device); err != nil {
		return fmt.Errorf("failed to format USB: %v", err)
	}
	return nil
}

// checkUSBSpace checks available space on the USB
func checkUSBSpace(usbMount string) (int64, error) {
	cmd := exec.Command("df", "-k", usbMount)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to check USB space: %v, output: %s", err, string(output))
	}
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("invalid df output: %s", string(output))
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return 0, fmt.Errorf("invalid df fields: %s", lines[1])
	}
	availableKB, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse available space: %v", err)
	}
	availableBytes := availableKB * 1024
	fmt.Printf("Available space on USB: %d MB\n", availableBytes/(1024*1024))
	return availableBytes, nil
}

// mountISO mounts the ISO and returns the mount point
func mountISO(isoPath string) (string, error) {
	fmt.Println("Mounting ISO...")
	cmd := exec.Command("hdiutil", "mount", isoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to mount ISO: %v, output: %s", err, string(output))
	}
	// Extract mount point from output (e.g., /Volumes/CCCOMA_X64FRE_EN-GB_DV9)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		for _, field := range fields {
			if strings.HasPrefix(field, "/Volumes/") {
				// Verify the mount point exists
				if _, err := os.Stat(field); os.IsNotExist(err) {
					return "", fmt.Errorf("mount point %s does not exist", field)
				}
				fmt.Printf("ISO mounted at: %s\n", field)
				return field, nil
			}
		}
	}
	return "", fmt.Errorf("could not find ISO mount point in output: %s", string(output))
}

// copyFiles copies ISO contents to USB, handling install.wim separately
func copyFiles(isoMount, usbMount string) error {
	fmt.Println("Copying ISO files to USB (excluding install.wim)...")
	// Ensure source path ends with a slash
	sourcePath := filepath.Clean(isoMount) + "/"
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path %s does not exist", sourcePath)
	}
	// Copy all files except sources/install.wim
	if err := runCommand("rsync", "-avh", "--progress", "--exclude", "sources/install.wim", sourcePath, usbMount+"/"); err != nil {
		return fmt.Errorf("failed to copy files: %v", err)
	}
	return nil
}

// splitWim splits install.wim into .swm files using wimlib
func splitWim(isoMount, usbMount string) error {
	wimPath := filepath.Join(isoMount, "sources", "install.wim")
	swmPath := filepath.Join(usbMount, "sources", "install.swm")
	fmt.Println("Splitting install.wim...")
	if _, err := os.Stat(wimPath); os.IsNotExist(err) {
		return fmt.Errorf("install.wim not found at %s", wimPath)
	}
	if err := runCommand("wimlib-imagex", "split", wimPath, swmPath, "4000"); err != nil {
		return fmt.Errorf("failed to split install.wim: %v", err)
	}
	return nil
}

// unmountVolumes unmounts the ISO and USB
func unmountVolumes(isoMount, usbMount string) error {
	fmt.Println("Unmounting ISO...")
	if err := runCommand("hdiutil", "unmount", isoMount); err != nil {
		fmt.Printf("Warning: failed to unmount ISO: %v\n", err)
	}
	fmt.Println("Ejecting USB...")
	if err := runCommand("diskutil", "eject", usbMount); err != nil {
		fmt.Printf("Warning: failed to eject USB: %v\n", err)
	}
	return nil
}

func main() {
	fmt.Println("Windows 11 Bootable USB Creator for Mac M1")
	fmt.Println("==========================================")
	fmt.Println("Ensure wimlib is installed (`brew install wimlib`) and the USB is inserted.")

	// Get ISO path
	isoPath, err := getISOPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get USB device
	device, err := getUSBDevice()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Confirm before formatting
	fmt.Printf("WARNING: This will erase all data on %s. Continue? (y/N): ", device)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
		fmt.Println("Aborted.")
		os.Exit(0)
	}

	// Format USB
	usbName := "WINUSB"
	if err := formatUSB(device, usbName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	usbMount := filepath.Join("/Volumes", usbName)

	// Check USB available space
	availableBytes, err := checkUSBSpace(usbMount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(availableBytes)

	// Mount ISO
	isoMount, err := mountISO(isoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Copy files (excluding install.wim)
	if err := copyFiles(isoMount, usbMount); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Split install.wim
	if err := splitWim(isoMount, usbMount); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Unmount volumes
	if err := unmountVolumes(isoMount, usbMount); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Success! The USB is now bootable with Windows 11.")
	fmt.Println("Insert it into your ASUS ROG Strix B850-F, set UEFI boot mode, and install.")
}