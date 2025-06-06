package config

// Config holds the application configuration
type Config struct {
	RepoPath     string
	Port         string
	Packages     []string
	ConfigFile   string
	Architecture string
	Distribution string
}
