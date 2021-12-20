package types

type VersionedFileReader interface {
	MajorVersions() []string
	Versions(major string) []string

	ReadFile(path, version string) ([]byte, error)
}

type Package struct {
	Source     string `yaml:"source"`
	TargetName string `yaml:"targetName"`
	Summary    string `yaml:"summary"`

	// This holds the major versions of the package (v0, v1, v2, ..), the fine versions are retrieved
	// with FileReader.Versions(majorVersion).
	Versions []string

	FileReader VersionedFileReader `yaml:"-"`
}
