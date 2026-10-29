package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"tripplanner/backend/internal/config"
)

func TestLoadDotEnvReadsFileWithoutOverridingProcessEnvironment(t *testing.T) {
	const fileValueKey = "TRIPPLANNER_DOTENV_TEST_FILE"
	const processValueKey = "TRIPPLANNER_DOTENV_TEST_PROCESS"
	previous, hadPrevious := os.LookupEnv(fileValueKey)
	_ = os.Unsetenv(fileValueKey)
	t.Cleanup(func() {
		if hadPrevious {
			_ = os.Setenv(fileValueKey, previous)
		} else {
			_ = os.Unsetenv(fileValueKey)
		}
	})
	t.Setenv(processValueKey, "from-process")
	path := filepath.Join(t.TempDir(), "backend.env")
	if err := os.WriteFile(path, []byte(fileValueKey+"=from-file\n"+processValueKey+"=from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadDotEnv(path); err != nil {
		t.Fatal("LoadDotEnv returned an error")
	}
	if got := os.Getenv(fileValueKey); got != "from-file" {
		t.Fatal("dotenv file value was not loaded")
	}
	if got := os.Getenv(processValueKey); got != "from-process" {
		t.Fatal("dotenv file replaced process environment")
	}
}
