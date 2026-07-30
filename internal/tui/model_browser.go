package tui

// BrowserModel holds file-browser vs terminal mode state for the main panel.
// Extracted from App to keep the top-level struct focused on orchestration.
type BrowserModel struct {
	FileBrowser *FileBrowser
	Active      bool
}
