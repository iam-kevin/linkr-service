package service

type RequestLinkCreate struct {
	// url to redirect to
	Url string `json:"redirect_url" validate:"required"`
	// if defined, the namespace redirect belongs
	Namespace string `json:"namespace,omitempty"`
	// if defined, how long the URL should be alive for
	ExpiresIn string `json:"empires_int,omitempty"`
}

type ResponseLinkCreate struct {
	ShortenedUrl     string `json:"short_url"`
	Identifier       string `json:"identifier"`
	Namespace        string `json:"namespace,omitempty"`
	ExpiresInSeconds *int64 `json:"expires_in_seconds,omitempty"`
	CreatedAt        string `json:"created_at"`
	ExpiresAt        string `json:"expires_at,omitempty"`
}
