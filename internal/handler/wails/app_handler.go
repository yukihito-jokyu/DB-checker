package wails

type AppHandler struct{}

type StatusResponse struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Version string `json:"version"`
}

func NewAppHandler() *AppHandler {
	return &AppHandler{}
}

func (h *AppHandler) Status() StatusResponse {
	return StatusResponse{
		Name:    "DB-checker",
		Ready:   true,
		Version: "dev",
	}
}
