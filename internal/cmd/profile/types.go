package profile

type profileListItem struct {
	Name     string `json:"name"`
	ClientID string `json:"client_id"`
	Current  bool   `json:"current"`
}

type currentProfile struct {
	Name string `json:"name"`
}

type profileDetail struct {
	Name         string `json:"name"`
	ClientID     string `json:"client_id"`
	Current      bool   `json:"current"`
	SecretStored bool   `json:"secret_stored"`
}
