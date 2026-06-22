package wails

type AppHandler struct{}

type StatusData struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Version string `json:"version"`
}

func NewAppHandler() *AppHandler {
	return &AppHandler{}
}

func (h *AppHandler) Status() Response[StatusData] {
	return OK(StatusData{
		Name:    "DB-checker",
		Ready:   true,
		Version: "dev",
	})
}
