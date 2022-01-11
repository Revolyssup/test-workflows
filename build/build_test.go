package build

import (
	"testing"
)

func TestCreateComponents(t *testing.T) {

	err := CreateComponents(StaticCompConfig{
		URL:     DefaultGenerationURL,
		Method:  DefaultGenerationMethod,
		Path:    workloadPath,
		DirName: LatestVersion,
		Config:  NewConfig(LatestVersion),
	})
	if err != nil {
		t.Fatalf("Failed to generate components: %s", err.Error())
	}
}
