package restapp

// APIEnvelope is how all REST resposes are wrapped
type APIEnvelope struct {
	Status  string
	Msg     string
	Payload interface{}
}
