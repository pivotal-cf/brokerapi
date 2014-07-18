package api

type Service struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Bindable    bool            `json:"bindable"`
	Plans       []ServicePlan   `json:"plans"`
	Metadata    ServiceMetadata `json:"metadata"`
	Tags        []string        `json:"tags"`
}

type ServicePlan struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Metadata    ServicePlanMetadata `json:"metadata"`
}

type ServicePlanMetadata struct {
	Bullets     []string                  `json:"bullets"`
	DisplayName string                    `json:"displayName"`
	Costs       []ServicePlanMetadataCost `json:"costs"`
}

type ServicePlanMetadataCost struct {
	Amount map[string]float64 `json:"amount"`
	Unit   string             `json:"unit"`
}

type ServiceMetadata struct {
	DisplayName      string                  `json:"displayName"`
	LongDescription  string                  `json:"longDescription"`
	DocumentationUrl string                  `json:"documentationUrl"`
	SupportUrl       string                  `json:"supportUrl"`
	Listing          ServiceMetadataListing  `json:"listing"`
	Provider         ServiceMetadataProvider `json:"provider"`
}

type ServiceMetadataListing struct {
	Blurb    string `json:"blurb"`
	ImageUrl string `json:"imageUrl"`
}

type ServiceMetadataProvider struct {
	Name string `json:"name"`
}
