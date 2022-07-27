package zapcloudwatch

import (
	. "github.com/onsi/ginkgo/v2"
	// . "github.com/onsi/gomega"
)

var _ = Describe("syncer", func() {
	Describe("start", func() {
		It("should return NotFound error when log stream not in group")
		It("pass on any other error when failing to describe")
	})

	Describe("sync", func() {
		It("should work with a highly concurrent sync/write")
	})

	Describe("stop", func() {
		It("should not leak any go-routines")
		It("should sync any data left in the buffer")
	})
})
