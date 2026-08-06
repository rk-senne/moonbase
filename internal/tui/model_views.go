package tui

// ViewsModel aggregates the per-view / panel / overlay state that App delegates to.
type ViewsModel struct {
	Dashboard   DashboardModel
	Pipeline    PipelineModel
	Terminal    TerminalModel
	Browser     BrowserModel
	Comms       CommsModel
	Mission     MissionModel
	Search      SearchModel
	SnippetPick SnippetPickerModel
	CtxFile     ContextFileModel
	Docs        *DocsState
	ProjectNav  *ProjectsState
	Tools       ToolsModel
	Settings    SettingsModel
}
