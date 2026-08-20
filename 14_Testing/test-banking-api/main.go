// Runs the bank HTTP API for manual exploration. The actual point of this
// project is the test suite in bank/ — run `go test ./...` from this
// directory to see it. This file just lets you poke at the same API with
// curl if you want to.
package main

import (
	"fmt"
	"log"
	"net/http"

	"testbankingapi/bank"
)

func main() {
	b := bank.NewBank(nil)
	b.OpenAccount("acc1", 100)
	b.OpenAccount("acc2", 50)

	fmt.Println("Bank API listening on http://localhost:8080")
	fmt.Println("Try:")
	fmt.Println(`  curl -X POST http://localhost:8080/deposit -d '{"account":"acc1","amount":25}'`)
	fmt.Println(`  curl -X POST http://localhost:8080/withdraw -d '{"account":"acc1","amount":200}'`)
	fmt.Println(`  curl "http://localhost:8080/balance?account=acc1"`)

	log.Fatal(http.ListenAndServe(":8080", bank.NewHTTPHandler(b)))
}
