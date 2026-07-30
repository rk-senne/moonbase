package tui

// cycleTheme advances to the next theme in the registry and recomputes styles.
// This is a pure value transformation on the model: it sets fields on the receiver
// and does NOT mutate any package-level state.
func (a *App) cycleTheme() {
	next := NextTheme(a.theme.Data)
	a.theme.Data = next
	a.theme.Name = next.Name
	a.theme.Styles = NewStyles(next)
}
