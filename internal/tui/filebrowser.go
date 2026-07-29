package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type FileEntry struct {
	Name  string
	IsDir bool
	Size  int64
}

type FileBrowser struct {
	dir     string
	entries []FileEntry
	cursor  int
	preview string
}

func newFileBrowser() *FileBrowser {
	cwd, _ := os.Getwd()
	fb := &FileBrowser{dir: cwd}
	fb.refresh()
	return fb
}

func (fb *FileBrowser) refresh() {
	entries, _ := os.ReadDir(fb.dir)
	fb.entries = nil
	// Directories first, then files
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue // skip hidden
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		fb.entries = append(fb.entries, FileEntry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
			Size:  size,
		})
	}
	// Sort: dirs first
	dirs := []FileEntry{}
	files := []FileEntry{}
	for _, e := range fb.entries {
		if e.IsDir {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	fb.entries = append(dirs, files...)
	if fb.cursor >= len(fb.entries) {
		fb.cursor = max(0, len(fb.entries)-1)
	}
	fb.updatePreview()
}

func (fb *FileBrowser) updatePreview() {
	fb.preview = ""
	if fb.cursor >= len(fb.entries) {
		return
	}
	entry := fb.entries[fb.cursor]
	if entry.IsDir {
		// Show dir contents as preview
		path := filepath.Join(fb.dir, entry.Name)
		items, _ := os.ReadDir(path)
		var lines []string
		for i, item := range items {
			if i >= 15 {
				lines = append(lines, fmt.Sprintf("  ... +%d more", len(items)-15))
				break
			}
			name := item.Name()
			if item.IsDir() {
				name += "/"
			}
			lines = append(lines, "  "+name)
		}
		fb.preview = strings.Join(lines, "\n")
	} else {
		// Show file contents
		path := filepath.Join(fb.dir, entry.Name)
		data, err := os.ReadFile(path)
		if err != nil {
			fb.preview = "  (cannot read)"
			return
		}
		lines := strings.Split(string(data), "\n")
		if len(lines) > 20 {
			lines = lines[:20]
			lines = append(lines, fmt.Sprintf("  ... (%d more lines)", len(strings.Split(string(data), "\n"))-20))
		}
		fb.preview = strings.Join(lines, "\n")
	}
}

func (fb *FileBrowser) Up() {
	if fb.cursor > 0 {
		fb.cursor--
		fb.updatePreview()
	}
}

func (fb *FileBrowser) Down() {
	if fb.cursor < len(fb.entries)-1 {
		fb.cursor++
		fb.updatePreview()
	}
}

func (fb *FileBrowser) Enter() {
	if fb.cursor >= len(fb.entries) {
		return
	}
	entry := fb.entries[fb.cursor]
	if entry.IsDir {
		fb.dir = filepath.Join(fb.dir, entry.Name)
		os.Chdir(fb.dir)
		fb.cursor = 0
		fb.refresh()
	}
}

func (fb *FileBrowser) Back() {
	fb.dir = filepath.Dir(fb.dir)
	os.Chdir(fb.dir)
	fb.cursor = 0
	fb.refresh()
}

func (fb *FileBrowser) SelectedPath() string {
	if fb.cursor >= len(fb.entries) {
		return ""
	}
	return filepath.Join(fb.dir, fb.entries[fb.cursor].Name)
}

func (fb *FileBrowser) SelectedIsFile() bool {
	if fb.cursor >= len(fb.entries) {
		return false
	}
	return !fb.entries[fb.cursor].IsDir
}

// Render the file browser for the main panel
func (a App) renderFileBrowser(width, maxH int) string {
	fb := a.fileBrowser
	borderColor := a.themeData.Active

	var s strings.Builder

	// Title with path
	cwdShort := fb.dir
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(cwdShort, home) {
		cwdShort = "~" + cwdShort[len(home):]
	}
	title := lipgloss.NewStyle().Foreground(a.themeData.Brand).Bold(true).Render("─ KND ")
	path := lipgloss.NewStyle().Foreground(a.themeData.Dim).Render(cwdShort + " ")
	titleW := lipgloss.Width(title)
	pathW := lipgloss.Width(path)
	fillW := max(1, width-titleW-pathW)
	s.WriteString(title + path + strings.Repeat("─", fillW) + "\n")

	// Split: file list left, preview right
	listW := width / 2
	previewW := width - listW - 2

	// File list
	var list strings.Builder
	maxFiles := maxH - 4
	if maxFiles < 3 {
		maxFiles = 3
	}
	start := 0
	if fb.cursor >= maxFiles {
		start = fb.cursor - maxFiles + 1
	}

	for i := start; i < len(fb.entries) && i < start+maxFiles; i++ {
		entry := fb.entries[i]
		icon := fileIcon(entry.Name, entry.IsDir)
		name := entry.Name
		if entry.IsDir {
			name += "/"
		}

		// Account for emoji icon width (2 cells) + prefix (" ▸ " = 4 cells) + space
		iconW := lipgloss.Width(icon)
		maxNameW := listW - iconW - 5
		if maxNameW < 4 {
			maxNameW = 4
		}
		if lipgloss.Width(name) > maxNameW {
			runes := []rune(name)
			for len(runes) > 0 && lipgloss.Width(string(runes)) > maxNameW {
				runes = runes[:len(runes)-1]
			}
			name = string(runes)
		}

		if i == fb.cursor {
			style := lipgloss.NewStyle().Foreground(a.themeData.Active).Bold(true)
			list.WriteString(style.Render(fmt.Sprintf(" ▸ %s %s", icon, name)) + "\n")
		} else {
			style := lipgloss.NewStyle().Foreground(a.themeData.Dim)
			if entry.IsDir {
				style = lipgloss.NewStyle().Foreground(a.themeData.Info)
			}
			list.WriteString(style.Render(fmt.Sprintf("   %s %s", icon, name)) + "\n")
		}
	}

	// Preview
	var prev strings.Builder
	prevTitle := lipgloss.NewStyle().Foreground(a.themeData.Info).Render("─ PREVIEW ─")
	prev.WriteString(prevTitle + "\n")
	if fb.preview != "" {
		lines := strings.Split(fb.preview, "\n")
		for i, line := range lines {
			if i >= maxH-4 {
				break
			}
			if len(line) > previewW-2 {
				line = line[:previewW-2]
			}
			prev.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Dim).Render(line) + "\n")
		}
	}

	listPanel := list.String()
	prevPanel := prev.String()
	body := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, " │ ", prevPanel)
	s.WriteString(body)

	// Footer
	s.WriteString("\n")
	count := fmt.Sprintf(" %d items", len(fb.entries))
	s.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Dim).Render(count))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Height(maxH).
		Render(s.String())
}

func fileIcon(name string, isDir bool) string {
	if isDir {
		return "📁"
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return "🔹"
	case ".js", ".ts", ".tsx":
		return "🟡"
	case ".java":
		return "☕"
	case ".md":
		return "📄"
	case ".json":
		return "📋"
	case ".yaml", ".yml":
		return "⚙️"
	case ".sh":
		return "🔧"
	case ".mod", ".sum":
		return "📦"
	default:
		return "  "
	}
}
