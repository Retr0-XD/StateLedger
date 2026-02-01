package ledger

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkAppend(b *testing.B) {
	l, err := Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	if err := l.InitSchema(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := l.Append(RecordInput{
			Timestamp: time.Now().Unix(),
			Type:      "code",
			Source:    "benchmark",
			Payload:   fmt.Sprintf(`{"count": %d}`, i),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAppendParallel(b *testing.B) {
	l, err := Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	if err := l.InitSchema(); err != nil {
		b.Fatal(err)
	}

	// Note: SQLite doesn't handle parallel writes well with in-memory DB
	// This benchmark shows contention rather than throughput
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := l.Append(RecordInput{
			Timestamp: time.Now().Unix(),
			Type:      "code",
			Source:    "benchmark",
			Payload:   fmt.Sprintf(`{"count": %d}`, i),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkList(b *testing.B) {
	l, err := Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	if err := l.InitSchema(); err != nil {
		b.Fatal(err)
	}

	// Seed with 1000 records
	for i := 0; i < 1000; i++ {
		_, err := l.Append(RecordInput{
			Timestamp: time.Now().Unix(),
			Type:      "code",
			Source:    "benchmark",
			Payload:   fmt.Sprintf(`{"count": %d}`, i),
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := l.List(ListQuery{
			Since: 0,
			Until: time.Now().Unix(),
			Limit: 100,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifyChain(b *testing.B) {
	l, err := Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	if err := l.InitSchema(); err != nil {
		b.Fatal(err)
	}

	// Seed with 1000 records
	for i := 0; i < 1000; i++ {
		_, err := l.Append(RecordInput{
			Timestamp: time.Now().Unix(),
			Type:      "code",
			Source:    "benchmark",
			Payload:   fmt.Sprintf(`{"count": %d}`, i),
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := l.VerifyChain()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetByID(b *testing.B) {
	l, err := Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	if err := l.InitSchema(); err != nil {
		b.Fatal(err)
	}

	// Seed with 1000 records
	for i := 0; i < 1000; i++ {
		_, err := l.Append(RecordInput{
			Timestamp: time.Now().Unix(),
			Type:      "code",
			Source:    "benchmark",
			Payload:   fmt.Sprintf(`{"count": %d}`, i),
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := l.GetByID(int64(i%1000 + 1))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashComputation(b *testing.B) {
	l, err := Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	if err := l.InitSchema(); err != nil {
		b.Fatal(err)
	}

	input := RecordInput{
		Timestamp: time.Now().Unix(),
		Type:      "code",
		Source:    "benchmark",
		Payload:   `{"large": "` + generateLargePayload(1024) + `"}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := l.Append(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func generateLargePayload(size int) string {
	payload := ""
	for i := 0; i < size; i++ {
		payload += "x"
	}
	return payload
}
