package tui

// ThemeModel aggregates the active theme: its name, the resolved colour palette
// (Data), and the derived Lip Gloss styles.
type ThemeModel struct {
	Name   string
	Data   Theme
	Styles Styles
}
