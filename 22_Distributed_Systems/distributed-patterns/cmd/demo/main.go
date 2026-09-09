// Distributed Patterns — a runnable demo of Event Sourcing, CQRS, the
// Saga Pattern, the Outbox Pattern, Distributed Locks, Leader Election,
// and Replication, each made directly OBSERVABLE rather than just
// described: you'll see a stale read model before a projector catches
// up, a saga's compensations actually run in reverse order, a duplicate
// webhook-style redelivery handled idempotently, a lock legitimately
// blocking a second holder, a leader failing over after a real TTL
// expiry, and a follower serving stale data immediately after an async
// write returns.
//
// Run with: go run ./cmd/demo
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"distributedpatterns/internal/cqrs"
	"distributedpatterns/internal/distributedlock"
	"distributedpatterns/internal/eventsourcing"
	"distributedpatterns/internal/leaderelection"
	"distributedpatterns/internal/outbox"
	"distributedpatterns/internal/replication"
	"distributedpatterns/internal/saga"
)

func section(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", 70))
}

func main() {
	section("1. Event Sourcing")
	store := eventsourcing.NewEventStore()
	store.Append("acc-1", 0, eventsourcing.AccountOpened{AccountID: "acc-1", Owner: "ada", At: time.Now()})
	store.Append("acc-1", 1, eventsourcing.MoneyDeposited{AccountID: "acc-1", Amount: 500, At: time.Now()})
	store.Append("acc-1", 2, eventsourcing.MoneyWithdrawn{AccountID: "acc-1", Amount: 120, At: time.Now()})

	events := store.Load("acc-1")
	state := eventsourcing.Replay(events)
	fmt.Printf("replayed %d events -> balance: %d (version %d)\n", len(events), state.Balance, state.Version)

	if err := store.Append("acc-1", 1, eventsourcing.MoneyDeposited{AccountID: "acc-1", Amount: 999}); err != nil {
		fmt.Println("appending with a STALE expected version correctly rejected:", err)
	}

	section("2. CQRS — the read model lags the write model, visibly")
	cmdHandler := cqrs.NewCommandHandler(store)
	readModel := cqrs.NewReadModel()
	projector := cqrs.NewProjector(store, readModel)

	cmdHandler.OpenAccount("acc-2", "bob")
	cmdHandler.Deposit("acc-2", 1000)

	if _, ok := readModel.GetBalance("acc-2"); !ok {
		fmt.Println("read model has NOT seen acc-2 yet — the projector hasn't caught up")
	}

	projector.CatchUp([]string{"acc-1", "acc-2"})
	bobBalance, _ := readModel.GetBalance("acc-2")
	fmt.Printf("after projector.CatchUp(), read model shows bob's balance: %d\n", bobBalance.Balance)

	cmdHandler.Withdraw("acc-2", 200)
	staleBalance, _ := readModel.GetBalance("acc-2")
	fmt.Printf("bob withdrew 200, but the read model STILL shows: %d (stale, until the next CatchUp)\n", staleBalance.Balance)
	projector.CatchUp([]string{"acc-2"})
	freshBalance, _ := readModel.GetBalance("acc-2")
	fmt.Printf("after another CatchUp: %d (now consistent)\n", freshBalance.Balance)

	section("3. Saga Pattern — happy path")
	orders := saga.NewOrderService()
	payments := saga.NewPaymentService()
	inventory := saga.NewInventoryService(map[string]int{"widget": 10})

	steps := saga.BuildOrderSaga(orders, payments, inventory, "order-1", "widget", 3, 4999)
	log, err := saga.Run(context.Background(), steps)
	for _, line := range log {
		fmt.Println(" ", line)
	}
	fmt.Println("saga result:", err, "| order status:", orders.Status("order-1"))

	section("4. Saga Pattern — failure triggers compensations, in reverse")
	steps2 := saga.BuildOrderSaga(orders, payments, inventory, "order-2", "widget", 999, 4999) // way more than the 7 remaining in stock
	log2, err2 := saga.Run(context.Background(), steps2)
	for _, line := range log2 {
		fmt.Println(" ", line)
	}
	fmt.Println("saga result:", err2, "| order status:", orders.Status("order-2"), "| was ever charged:", payments.WasCharged("order-2"))

	section("5. Outbox Pattern — atomic write, at-least-once delivery, and a real redelivery")
	obStore := outbox.NewStore()
	var delivered []string
	relay := outbox.NewRelay(obStore, func(eventType, payload string) error {
		delivered = append(delivered, fmt.Sprintf("%s: %s", eventType, payload))
		return nil
	})

	obStore.PlaceOrderWithEvent("order-100", `{"orderID":"order-100","total":4999}`)
	n, _ := relay.Poll()
	fmt.Printf("relay delivered %d new event(s): %v\n", n, delivered)

	obStore.PlaceOrderWithEvent("order-101", `{"orderID":"order-101","total":1999}`)
	relay.SimulateCrashBeforeMarkingSent() // delivers it, but "crashes" before marking sent
	fmt.Println("simulated a crash right after delivering order-101's event, before marking it sent")
	n2, _ := relay.Poll() // on restart, the relay sees it as still unsent and redelivers
	fmt.Printf("next Poll() redelivers it: %d event(s) — total delivered log: %v\n", n2, delivered)
	fmt.Println("(order-101's event appears TWICE — exactly Module 20/21's idempotent-consumer")
	fmt.Println(" requirement: a real subscriber must dedupe this by event ID, same as a webhook)")

	section("6. Distributed Locks")
	lockStore := distributedlock.NewLockStore()
	gotA := lockStore.TryLock("daily-report-job", "worker-A", 200*time.Millisecond)
	gotB := lockStore.TryLock("daily-report-job", "worker-B", 200*time.Millisecond)
	fmt.Println("worker-A acquired the lock:", gotA)
	fmt.Println("worker-B tried immediately after, acquired:", gotB, "(blocked — A already holds it)")

	time.Sleep(250 * time.Millisecond) // let A's lease expire
	gotB2 := lockStore.TryLock("daily-report-job", "worker-B", 200*time.Millisecond)
	fmt.Println("worker-B tries again after A's lease expired, acquired:", gotB2)

	unlocked := lockStore.Unlock("daily-report-job", "worker-A")
	fmt.Println("worker-A (no longer the actual holder) tries to Unlock anyway. Succeeded:", unlocked)
	fmt.Println("(correctly false — Unlock checks who ACTUALLY holds the lock before releasing it,")
	fmt.Println(" preventing worker-A from accidentally releasing worker-B's now-valid lock)")

	section("7. Leader Election")
	leaseStore := leaderelection.NewLeaseStore()
	nodeA := leaderelection.NewNode("node-A", leaseStore, 150*time.Millisecond)
	nodeB := leaderelection.NewNode("node-B", leaseStore, 150*time.Millisecond)

	go nodeA.Run(40*time.Millisecond, func(id string) { fmt.Println(" ", id, "became leader") }, func(id string) { fmt.Println(" ", id, "lost leadership") })
	go nodeB.Run(40*time.Millisecond, func(id string) { fmt.Println(" ", id, "became leader") }, func(id string) { fmt.Println(" ", id, "lost leadership") })

	time.Sleep(200 * time.Millisecond)
	leader, _ := leaseStore.CurrentLeader()
	fmt.Println("current leader after both nodes have been running:", leader)

	fmt.Println("stopping the current leader (simulating a crash)...")
	if leader == "node-A" {
		nodeA.Stop()
	} else {
		nodeB.Stop()
	}
	time.Sleep(300 * time.Millisecond) // give the lease time to expire and the survivor time to notice
	newLeader, _ := leaseStore.CurrentLeader()
	fmt.Println("new leader after failover:", newLeader)
	nodeA.Stop()
	nodeB.Stop()

	section("8. Replication — sync vs. async, timed and observed directly")
	slowFollower := &replication.Follower{Name: "follower-slow", Latency: 150 * time.Millisecond}
	fastFollower := &replication.Follower{Name: "follower-fast", Latency: 5 * time.Millisecond}
	leaderNode := replication.NewLeader(slowFollower, fastFollower)

	syncDuration := leaderNode.WriteSync("entry-1")
	fmt.Printf("WriteSync took %v (waited for the SLOWEST follower)\n", syncDuration.Round(time.Millisecond))
	fmt.Println("slow follower's log immediately after:", slowFollower.Read(), "(already has it — sync waited)")

	asyncDuration := leaderNode.WriteAsync("entry-2")
	fmt.Printf("WriteAsync took %v (returned immediately)\n", asyncDuration.Round(time.Millisecond))
	fmt.Println("slow follower's log IMMEDIATELY after:", slowFollower.Read(), "(entry-2 not there yet — STALE READ)")
	time.Sleep(200 * time.Millisecond)
	fmt.Println("slow follower's log 200ms later:          ", slowFollower.Read(), "(caught up — EVENTUALLY consistent)")

	fmt.Println()
	fmt.Println("Demo complete. See README.md for a diagrammed walkthrough of every")
	fmt.Println("pattern exercised above.")
}
