package bank

import "testing"

// BenchmarkAccount_Deposit measures Deposit's steady-state performance.
// b.N is chosen automatically by the testing framework to get a stable
// measurement — see Module 14's guide, section 3.
func BenchmarkAccount_Deposit(b *testing.B) {
	acc := NewAccount("acc1", 0, nil)
	b.ResetTimer() // exclude the setup above from the measured time

	for i := 0; i < b.N; i++ {
		acc.Deposit(1.0)
	}
}

// BenchmarkAccount_Withdraw resets the account EVERY iteration, since
// Withdraw would otherwise start failing (insufficient funds) partway
// through a long run — the setup cost of NewAccount is deliberately
// excluded from measurement via b.StopTimer()/b.StartTimer() around it.
func BenchmarkAccount_Withdraw(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		acc := NewAccount("acc1", 1000000, nil) // plenty of balance for any b.N
		b.StartTimer()

		acc.Withdraw(1.0)
	}
}

// BenchmarkParseAmount measures the parser's cost — a good candidate to run
// with -benchmem, since ParseAmount does string manipulation (TrimSpace,
// TrimPrefix) that may or may not allocate depending on the input.
func BenchmarkParseAmount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseAmount("$19.99")
	}
}

// BenchmarkDepositWithLogger compares against BenchmarkAccount_Deposit to
// show the (usually small) overhead a real logger implementation adds —
// run both together with `go test -bench=Deposit -benchmem` to compare.
func BenchmarkDepositWithLogger(b *testing.B) {
	logger := &FakeLogger{}
	acc := NewAccount("acc1", 0, logger)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		acc.Deposit(1.0)
	}
}
