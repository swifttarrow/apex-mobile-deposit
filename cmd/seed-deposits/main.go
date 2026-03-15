// seed-deposits inserts 25 deposits in FundsPosted state with created_at spread
// so that some fall before the EOD cutoff (6:30 PM CT) and some after. Use this
// to demo settlement batching and rollover without time travel.
//
// Usage: go run ./cmd/seed-deposits [--count=25] [--before=15] [--after=10]
// Default: 15 before cutoff (same settlement day), 10 after (next business day).
// Uses DATABASE_URL or checkstream.db.
package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"time"

	"github.com/checkstream/checkstream/internal/db"
	"github.com/google/uuid"
)

func main() {
	count := flag.Int("count", 25, "Total number of deposits to insert")
	before := flag.Int("before", 15, "Deposits with created_at before 6:30 PM CT (same settlement day)")
	after := flag.Int("after", 10, "Deposits with created_at after 6:30 PM CT (next business day)")
	flag.Parse()

	if *before+*after != *count {
		log.Fatalf("before + after must equal count: %d + %d != %d", *before, *after, *count)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "checkstream.db"
	}

	database, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		loc = time.FixedZone("CT", -6*3600)
	}

	now := time.Now().In(loc)
	// Use today in CT: 2:00 PM = before cutoff, 7:00 PM = after cutoff
	beforeTime := time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, loc)
	afterTime := time.Date(now.Year(), now.Month(), now.Day(), 19, 0, 0, 0, loc)
	beforeRFC := beforeTime.UTC().Format(time.RFC3339)
	afterRFC := afterTime.UTC().Format(time.RFC3339)

	accountID := "ACC-001"
	state := "FundsPosted"

	inserted := 0
	for i := 0; i < *before; i++ {
		id := uuid.New().String()
		amount := 50.0 + float64(i*10)
		createdAt := beforeRFC
		updatedAt := beforeRFC
		if err := insertTransfer(database, id, accountID, amount, state, createdAt, updatedAt); err != nil {
			log.Fatalf("insert before-cutoff transfer: %v", err)
		}
		inserted++
	}
	for i := 0; i < *after; i++ {
		id := uuid.New().String()
		amount := 75.0 + float64(i*15)
		createdAt := afterRFC
		updatedAt := afterRFC
		if err := insertTransfer(database, id, accountID, amount, state, createdAt, updatedAt); err != nil {
			log.Fatalf("insert after-cutoff transfer: %v", err)
		}
		inserted++
	}

	log.Printf("Inserted %d deposits: %d before 6:30 PM CT, %d after (next business day). Run settlement to see batch vs rollover.", inserted, *before, *after)
}

func insertTransfer(db *sql.DB, id, accountID string, amount float64, state, createdAt, updatedAt string) error {
	_, err := db.Exec(`
		INSERT INTO transfers (id, account_id, amount, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, accountID, amount, state, createdAt, updatedAt,
	)
	return err
}
