package tui

// muxChoice describes the terminal multiplexer selected for the host OS.
type muxChoice struct {
	Tool string   // "cmux" or "tmux"
	Bin  string   // resolved executable path
	Args []string // launch arguments (cmux only; tmux args are decided at launch)
}

// lookPathFunc mirrors exec.LookPath and is injectable for tests.
type lookPathFunc func(string) (string, error)

// selectMultiplexer picks the terminal multiplexer for the given OS:
//
//   - macOS ("darwin"): cmux is preferred (it is purpose-built for AI coding
//     agents), falling back to tmux when cmux is not installed.
//   - Linux and every other platform: tmux. cmux is macOS-only, so it is never
//     selected off macOS even if a binary named "cmux" happens to be on PATH.
//
// It returns ok=false when no supported multiplexer is installed.
func selectMultiplexer(goos string, look lookPathFunc) (muxChoice, bool) {
	tmux := func() (muxChoice, bool) {
		if bin, err := look("tmux"); err == nil {
			return muxChoice{Tool: "tmux", Bin: bin}, true
		}
		return muxChoice{}, false
	}

	if goos == "darwin" {
		if bin, err := look("cmux"); err == nil {
			return muxChoice{
				Tool: "cmux",
				Bin:  bin,
				Args: []string{"workspace", "new", "--name", "moonbase"},
			}, true
		}
		return tmux()
	}

	// Linux, *BSD, etc. — tmux only.
	return tmux()
}
