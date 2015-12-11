package brokerapi

type EmptyResponse struct{}

type ErrorResponse struct {
	Error       string `json:"error,omitempty"`
	Description string `json:"description"`
}

type CatalogResponse struct {
	Services []Service `json:"services"`
}

type ProvisioningResponse struct {
	DashboardURL string `json:"dashboard_url,omitempty"`
}

type LastOperationResponse struct {
	State       string `json:"state"`
	Description string `json:"description,omitempty"`
}
