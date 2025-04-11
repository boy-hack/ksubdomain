package result

type Result struct {
	Subdomain string   `json:"subdomain"`
	Answers   []string `json:"answers"`
}
