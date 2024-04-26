package service

type RequestClientCreate struct {
	// type of client accessing resource
	// options: admin | read-write | read-only | write-only
	Role string `json:"role,omitempty"`
}

type ResponseClientCreate struct {
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}
