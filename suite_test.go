package zapcloudwatch_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestZapcloudwatch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Zapcloudwatch Suite")
}
