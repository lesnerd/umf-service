package shared

// Home represents the service home directory structure
type Home interface {
	HomeDir() string
	LogDir() string
	DataDir() string
	ConfigDir() string
	SystemConfigFile() string
}