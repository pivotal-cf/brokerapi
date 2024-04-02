package blog_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBlog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BrokerAPI logger Suite")
}
