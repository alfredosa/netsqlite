# netsqlite

netsqlite is sqlite3 over the wire, with pools to manage your databases, with the added benefit of using authentication. It aims to do nothing more than that. 

I expect that eventually I will add a distributed nature to this, but sqlite3 aims to be simple and powerful at the same time, and I aim to keep this simple. 

Otherwise I would just use postgres 🐘. 

## Why?

I like sqlite3, and sometimes, the simplicity of it, having `1 file = 1 database`, the C code is pristine, and I wish I could interact with it over the wire. I can deploy this once and have multiple applications talk to sqlite3 safely, concurrently, and also importantly securely. 

Visualize all my dbs in one place and execute or maintain them, including backing them up. 

Sometimes I use kubernetes, and it makes it easy to just have one Volume to take care of for example

## Why not `<insert-name-of-cool-project>`

In my search I noticed that a lot of other projects either:

1. Are overly complicated
2. Do much more than just the above.

other projects you might consider:

- `rqlite` impressive, but for me it transformed the nature of sqlite, even has a different shell with custom commands.
- I took inspiration from `turso` but yet again, they redesigned sqlite3.

My aim was not to change sqlite, just interact with it over the wire, that it managed the connections through a pool, and that the encoding was not expensive.

Anything more than this requires a much more capable database, in my opinion.


## How do I use it, you say?

I made drivers so that you can just import them and use sqlite abstractions like you would with sqlite3 🏴‍☠️ 

``` go
package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/alfredosa/netsqlite/pkg/drivers" // Register the driver
)

func main() {
	// DSN: netsqlite://database_name:host:port/token
	// If 'mydatabase.db' doesn't exist, it will create it (with WAL Enabled, for concurrent usage). 
	dsn := "netsqlite://mydatabase.db:localhost:3541/SUPERSECRETTOKEN" // Use a valid token

	db, err := sql.Open("netsqlite", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Ping
	log.Println("Pinging server...")
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Ping failed: %v", err)
	}

	// 2. Exec (Create table and Insert)
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS items (id INTEGER PRIMARY KEY, name TEXT, price REAL)`)
	if err != nil {
		log.Fatalf("Exec (create) failed: %v", err)
	}
	result, err := db.ExecContext(ctx, `INSERT INTO items (name, price) VALUES (?, ?)`, "Gadget", 19.99)
	if err != nil {
		log.Fatalf("Exec (insert) failed: %v", err)
	}
	lid, _ := result.LastInsertId()
	aff, _ := result.RowsAffected()
	log.Printf("Insert successful: LastInsertID=%d, RowsAffected=%d", lid, aff)

    // Insert another
    _, err = db.ExecContext(ctx, `INSERT INTO items (name, price) VALUES (?, ?)`, "Widget", 5.45)
	if err != nil {
		log.Fatalf("Exec (insert 2) failed: %v", err)
	}


	// 3. Query
	rows, err := db.QueryContext(ctx, `SELECT id, name, price FROM items WHERE price > ? ORDER BY id`, 10.0)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id    int64
			name  string
			price float64
		}
		if err := rows.Scan(&id, &name, &price); err != nil {
			log.Fatalf("Scan failed: %v", err)
		}
		log.Printf("  - ID: %d, Name: %s, Price: %.2f\n", id, name, price)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("Row iteration error: %v", err)
	}
}
```

Like and subscribe for more content

### Notes and disclosures:

- This is primarily to solve a personal need. But I hope someone finds it helpful
