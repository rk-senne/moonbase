package tui

// EnvModel aggregates detected runtime/environment state: git+threat (System),
// host infra like the file watcher, tool cache and platform context (Infra),
// and the available/active AI backends (Backend).
type EnvModel struct {
	System  SystemModel
	Infra   InfraModel
	Backend BackendModel
}
