package api_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"code.google.com/p/go-uuid/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

func fixture(name string) string {
	filePath := path.Join("fixtures", name)
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("Could not read fixture: %s", name))
	}

	return string(contents)
}

func nullLogger() *log.Logger {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		panic("Could not make a null logger")
	}
	return log.New(devNull, "", 0)
}

func uniqueID() string {
	return uuid.NewRandom().String()
}

func uniqueInstanceID() string {
	return uniqueID()
}

func uniqueBindingID() string {
	return uniqueID()
}
