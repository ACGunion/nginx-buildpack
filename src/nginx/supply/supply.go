package supply

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Manifest interface {
	DefaultVersion(depName string) (libbuildpack.Dependency, error)
	AllDependencyVersions(string) []string
	InstallDependency(dep libbuildpack.Dependency, outputDir string) error
	RootDir() string
}
type Stager interface {
	AddBinDependencyLink(string, string) error
	DepDir() string
	BuildDir() string
}

type Config struct {
	Version string `yaml:"version"`
}

type Supplier struct {
	Stager       Stager
	Manifest     Manifest
	Log          *libbuildpack.Logger
	Config       Config
	VersionLines map[string]string
}

func New(stager Stager, manifest Manifest, logger *libbuildpack.Logger) *Supplier {
	return &Supplier{
		Stager:   stager,
		Manifest: manifest,
		Log:      logger,
	}
}

func (s *Supplier) Run() error {
	if err := s.Setup(); err != nil {
		return err
	}

	if err := s.InstallNginx(); err != nil {
		return err
	}

	return nil
}

func (s *Supplier) Setup() error {
	configPath := filepath.Join(s.Stager.BuildDir(), "nginx.yml")
	if exists, err := libbuildpack.FileExists(configPath); err != nil {
		return err
	} else if exists {
		if err := libbuildpack.NewYAML().Load(configPath, &s.Config); err != nil {
			return err
		}
	}

	var m struct {
		VersionLines map[string]string `yaml:"version_lines"`
	}
	if err := libbuildpack.NewYAML().Load(filepath.Join(s.Manifest.RootDir(), "manifest.yml"), &m); err != nil {
		return err
	}
	s.VersionLines = m.VersionLines

	return nil
}

func (s *Supplier) findMatchingVersion(depName string, version string) (libbuildpack.Dependency, error) {
	dir := filepath.Join(s.Stager.DepDir(), depName)
	if val, ok := s.VersionLines[version]; ok {
		version = val
	}

	versions := s.Manifest.AllDependencyVersions(depName)
	if ver, err := libbuildpack.FindMatchingVersion(version, versions); err != nil {
		return libbuildpack.Dependency{}, err
	} else {
		version = ver
	}

	return libbuildpack.Dependency{Name: depName, Version: version}, nil
}

func (s *Supplier) InstallNginx() error {
	dep := s.findMatchingVersion("nginx", s.Config.Version)
	s.Log.BeginStep("Requested nginx version: %s => %s", s.Config.Version, dep.Version)

	if err := s.Manifest.InstallDependency(dep, dir); err != nil {
		return fmt.Errorf("Could not install nginx: %s", err)
	}

	return s.Stager.AddBinDependencyLink(filepath.Join(dir, "nginx", "sbin", "nginx"), "nginx")
}
