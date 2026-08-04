# Project 1 — Inventory System

```bash
cd inventory-system
go run main.go
```

Add a couple of regular products and a perishable one, list everything, then
try adjusting quantity below zero to see the error path.

## What's Demonstrated Here

- **Embedding** — `Perishable` embeds `Product` (no field name), so
  `perishable.SKU`, `perishable.Name`, etc. are all **promoted** straight
  through, and `Perishable` adds its own `ExpiryDate` field and `IsExpired()`
  method that plain `Product` doesn't have.
- **Maps of pointers (`map[string]*Product`)** — storing `*Product` instead
  of `Product` means `adjustQuantity` can mutate the actual stored item
  through the map (`p.Quantity += delta`) without having to write the whole
  struct back into the map afterward.
- **Building a sorted view over multiple maps** — `printAll` and `lowStock`
  both collect keys from *two* maps into one `[]string`, sort it, then use
  that stable order to decide what to print — same pattern as the Banking
  Menu project in Module 02, just across two maps instead of one.

```
┌─────────────────────────────────────────────────────┐
│  type Product struct { SKU, Name, Price, Quantity }      │
│                                                              │
│  type Perishable struct {                                     │
│      Product        ◀── embedded, no field name                │
│      ExpiryDate time.Time                                         │
│  }                                                                   │
│                                                                         │
│  p := Perishable{...}                                                    │
│  p.SKU          ◀── promoted from Product, reads as if Perishable          │
│                      declared SKU itself                                     │
│  p.IsExpired()   ◀── Perishable's OWN method — Product has no idea            │
│                      this method exists at all                                  │
└─────────────────────────────────────────────────────┘
```

## Case Study: Why Two Separate Maps, Not One

It might seem simpler to store everything in one `map[string]Product` and
add an `IsPerishable bool` + `ExpiryDate time.Time` field directly on
`Product`. That works, but it means **every** product carries expiry fields
it'll never use, and nothing stops you from accidentally setting an expiry
date on a can of paint. Splitting into `Product` and `Perishable` (via
embedding) means the type system itself enforces the distinction — a
`Product` value can never have an `ExpiryDate`, because the field simply
doesn't exist on that type. The cost is exactly what you see here: code that
touches "any item" (like `totalValue` or `printAll`) has to check both maps.
For a small project like this, checking two maps is a fine trade for that
extra type safety; in a bigger system, you'd likely reach for a shared
**interface** instead (Module 06 covers those) so both types could live in
one collection.

## Try It Yourself
- Add a `removeProduct(sku string) error` that deletes from whichever map
  actually has that SKU
- Add a "restock all perishables expiring within N days" bulk action —
  practice ranging over `inv.perishables` with a conditional
- Change `lowStock` to also report *how much* stock (not just the SKU), by
  returning `[]string` formatted lines instead of bare SKUs
