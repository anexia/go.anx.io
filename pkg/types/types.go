package types

type VersionedFileReader interface {
	Versions() []string
	ReadFile(path, version string) ([]byte, error)
}

type Package struct {
	Source     string `yaml:"source"`
	TargetName string `yaml:"targetName"`
	Summary    string `yaml:"summary"`

	FileReader VersionedFileReader `yaml:"-"`
}
