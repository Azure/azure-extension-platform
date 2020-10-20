package handlerenv

// HandlerEnvironment describes the handler environment configuration for an extension
type HandlerEnvironment struct {
	HeartbeatFile string
	StatusFolder  string
	ConfigFolder  string
	LogFolder     string
	DataFolder    string
}
