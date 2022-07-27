package zapcloudwatch

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Buffer2", func() {
	var buf *Buffer
	BeforeEach(func() { buf = NewBuffer() })

	It("should push log events and batch", func() {
		var full, disc bool
		for i := int64(0); i < 100; i++ {
			full, disc = buf.Push(LogEvent(i, "foo"))
			Expect(full).To(BeFalse())
			Expect(disc).To(BeFalse())
		}

		Expect(buf.All()).To(HaveLen(100))

		// batch should reset the buffer state
		Expect(buf.Batch()).To(HaveLen(100))
		Expect(buf.All()).To(HaveLen(0))
	})

	It("batch should sort the events", func() {
		buf.Push(LogEvent(1, "A"))
		buf.Push(LogEvent(0, "B"))
		buf.Push(LogEvent(3, "C"))
		batch := buf.Batch()
		Expect(batch).To(HaveLen(3))

		Expect(*batch[0].Message).To(Equal("B"))
		Expect(*batch[1].Message).To(Equal("A"))
		Expect(*batch[2].Message).To(Equal("C"))
	})

	It("should push to a max of 10k", func() {
		var full, disc bool
		for i := int64(0); i < 10000; i++ {
			full, disc = buf.Push(LogEvent(i, "foo"))
			Expect(full).To(BeFalse())
			Expect(disc).To(BeFalse())
		}

		full, disc = buf.Push(LogEvent(10000, "foo"))
		Expect(full).To(BeTrue())
		Expect(disc).To(BeFalse())

		Expect(buf.All()).To(HaveLen(10000))
	})

	It("should push log that casuse span of buffer to be > 24 hours apart", func() {
		var full, disc bool
		full, disc = buf.Push(LogEvent(0, "A"))
		Expect(full).To(BeFalse())
		Expect(disc).To(BeFalse())

		full, disc = buf.Push(LogEvent(1, "B"))
		Expect(full).To(BeFalse())
		Expect(disc).To(BeFalse())

		// C is pushed after B, but it's timestamp is earlier. So early that it should split the batch
		full, disc = buf.Push(LogEvent(-86400000, "C"))
		Expect(full).To(BeTrue())
		Expect(disc).To(BeFalse())

		Expect(buf.All()).To(HaveLen(2))
	})

	It("should not push past the max byte size", func() {
		var full, disc bool
		for i := int64(0); i < 1024; i++ {
			full, disc = buf.Push(LogEvent(i, strings.Repeat("a", 998)))
			Expect(full).To(BeFalse())
			Expect(disc).To(BeFalse())
		}

		full, disc = buf.Push(LogEvent(1024, strings.Repeat("a", 999)))
		Expect(full).To(BeTrue())
		Expect(disc).To(BeFalse())

		Expect(buf.All()).To(HaveLen(1024))
	})

	It("should push events that are too far in the future", func() {
		var full, disc bool
		full, disc = buf.Push(LogEvent(0, "A"))
		Expect(full).To(BeFalse())
		Expect(disc).To(BeFalse())
		full, disc = buf.Push(LogEvent(7200000, "B"))
		Expect(full).To(BeFalse())
		Expect(disc).To(BeTrue())

		Expect(buf.All()).To(HaveLen(1))
	})

	It("should push events that are too far in the past", func() {
		full, disc := buf.Push(LogEvent(-1210000000, "A"))
		Expect(full).To(BeFalse())
		Expect(disc).To(BeTrue())

		Expect(buf.All()).To(HaveLen(0))
	})
})
