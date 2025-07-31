package response

type ImportServerResponse struct {
	ImportedCount   int      `json:"imported_count"`
	ImportedServers []string `json:"imported_servers,omitempty"`
	FailedCount     int      `json:"failed_count"`
	FailedServers   []string `json:"failed_servers,omitempty"`
}
