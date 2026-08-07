# Go for Beginners — Module 06: Methods & Interfaces

## Contents

1. **[06-methods-and-interfaces.md](./06-methods-and-interfaces.md)** — Methods
   and receivers, interfaces (implicit/structural satisfaction — no
   `implements` keyword), composition, embedding for shared behavior, type
   assertions, the empty interface (`any`), and reflection basics — with a
   reminder that reflection should stay rare in everyday code. Diagrams
   included throughout.

2. **[payment-gateway/](./payment-gateway)** — A `Gateway` that processes
   payments through any `PaymentProcessor`, without ever knowing which
   concrete type it holds. Introduces a *separate*, optional `Refundable`
   interface (crypto can't refund; cards and PayPal can), discovered via
   type assertion — plus `any` for freeform receipt metadata and a small
   `reflect`-based generic receipt printer.

3. **[notification-service/](./notification-service)** — `MultiNotifier`
   satisfies `Notifier` by *fanning out* to a slice of other `Notifier`s (the
   Composite pattern), including nesting a `MultiNotifier` inside another
   one for free. Puts a type assertion and a type switch side by side so the
   difference between them is concrete, not abstract.

4. **[storage-drivers/](./storage-drivers)** — `Store`, composed from three
   small interfaces (`Getter`/`Setter`/`Deleter`), with two backends:
   `MemoryStorage` (no cleanup needed) and `FileStorage` (buffers writes,
   flushes on `Close()`). `Closer` is genuinely optional — discovered via
   `closeIfPossible`'s type assertion — and a driver **registry** mirrors the
   standard library's own `database/sql` driver-by-name pattern.

## Suggested Order

```
Methods & Interfaces guide ──▶ Payment Gateway ──▶ Notification Service ──▶ Storage Drivers
```

All three projects repeat the same core move — a small interface, one or
more concrete types satisfying it implicitly, and a type assertion checking
for *optional* extra behavior — deliberately, so the pattern lands as a
reusable habit rather than a one-off trick. Each project applies it to a
different kind of "optional capability": refunds, validation, and cleanup.

## Quick Reference: Running Any Project

```bash
cd <project-folder>
go run main.go
```

*Note: this module builds on Modules 00–05 — start there first if you
haven't already, especially Module 04 (embedding) and Module 05 (pointer
receivers), which this module extends directly.*
