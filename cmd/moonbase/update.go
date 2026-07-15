package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/f5508037/moonbase/internal/updater"
)

// runUpdateCheck checks for available updates without installing.
func runUpdateCheck() {
	fmt.Println("🌙 Checking for updates...")
	fmt.Println()

	result, err := updater.CheckForUpdate(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		osExit(1)
	}

	if !result.UpdateNeeded {
		fmt.Printf("   ✅ You're on the latest version (%s)\n", result.CurrentVersion)
		return
	}

	fmt.Printf("   Current: v%s\n", result.CurrentVersion)
	fmt.Printf("   Latest:  v%s\n", result.LatestVersion)
	fmt.Println()

	if result.AssetName != "" {
		fmt.Printf("   Binary:  %s\n", result.AssetName)
	} else {
		fmt.Printf("   ⚠️  No binary available for %s/%s\n", runtime.GOOS, runtime.GOARCH)
	}

	if result.ReleaseURL != "" {
		fmt.Printf("   Release: %s\n", result.ReleaseURL)
	}

	fmt.Println()
	fmt.Println("   Run 'moonbase update' to install.")
}

// runUpdate downloads and installs the latest version.
func runUpdate() {
	fmt.Println("🌙 Moonbase Self-Update")
	fmt.Println()

	if version == "dev" {
		fmt.Fprintln(os.Stderr, "❌ Cannot update a development build.")
		fmt.Fprintln(os.Stderr, "   Build a release version first: make build VERSION=x.y.z")
		fmt.Fprintln(os.Stderr, "   Or install from GitHub Releases.")
		osExit(1)
	}

	// Check first
	fmt.Printf("   Current version: v%s\n", version)
	fmt.Printf("   Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()
	fmt.Println("   Checking GitHub Releases...")

	result, err := updater.CheckForUpdate(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		osExit(1)
	}

	if !result.UpdateNeeded {
		fmt.Printf("   ✅ Already at latest version (v%s)\n", result.CurrentVersion)
		return
	}

	fmt.Printf("   New version available: v%s → v%s\n", result.CurrentVersion, result.LatestVersion)
	fmt.Println()

	if result.AssetName == "" {
		fmt.Fprintf(os.Stderr, "❌ No binary available for %s/%s in this release.\n", runtime.GOOS, runtime.GOARCH)
		fmt.Fprintf(os.Stderr, "   Check: %s\n", result.ReleaseURL)
		osExit(1)
	}

	// Download and install
	fmt.Printf("   Downloading %s...\n", result.AssetName)

	newVersion, err := updater.Update(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Update failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "   Your current binary is unchanged.")
		osExit(1)
	}

	fmt.Println()
	fmt.Printf("   ✅ Updated to v%s\n", newVersion)
	fmt.Println()

	// Show release notes summary (first 5 lines)
	if result.ReleaseNotes != "" {
		fmt.Println("   Release notes:")
		printTruncated(result.ReleaseNotes, 8)
		fmt.Println()
	}

	fmt.Println("   Run 'moonbase version' to confirm.")
}

// printTruncated prints at most maxLines of text, indented.
func printTruncated(text string, maxLines int) {
	lines := 0
	start := 0
	for i, ch := range text {
		if ch == '\n' {
			if lines < maxLines {
				line := text[start:i]
				if line != "" {
					fmt.Printf("     %s\n", line)
				}
			}
			lines++
			start = i + 1
		}
	}
	// Last line (no trailing newline)
	if start < len(text) && lines < maxLines {
		fmt.Printf("     %s\n", text[start:])
		lines++
	}
	if lines >= maxLines {
		fmt.Println("     ...")
	}
}
