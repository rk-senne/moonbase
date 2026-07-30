package tui

// BootModel holds boot-sequence state — readiness flag and the current step
// in the animated startup sequence. Extracted from App to keep the top-level
// struct focused on orchestration.
type BootModel struct {
	Ready bool
	Step  int
}
