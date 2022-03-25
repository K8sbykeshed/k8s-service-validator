package entities

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEntities(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Entities Suite")
}
