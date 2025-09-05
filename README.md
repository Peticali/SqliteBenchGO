# SQLite WAL Benchmark Go

This project is a **SQLite benchmark** written in Go, designed to test the performance of concurrent reads and writes on a SQLite database configured with **WAL (Write-Ahead Logging)**. The benchmark allows you to configure the number of reader and writer threads, the duration of the test, and whether the database runs in memory or on disk.

The program also generates a PNG plot showing the evolution of `Writes/s` and `Reads/s` throughout the benchmark.

---

## Prerequisites

- Go >= 1.20
- Dependencies:

```bash
github.com/jmoiron/sqlx
github.com/mattn/go-sqlite3
gonum.org/v1/plot
```

## Usage

The program accepts CLI flags to configure the benchmark:

```bash
go run main.go -writers=2 -readers=10 -duration=10 -memory=false
```
| Flag | Tipo | Padrão	| Descrição |
| -------- | ------- | ------- | ------- |
| -writers | int | 2 | Number of writer threads | 
| -readers | int | 10 | Number of reader threads | 
| -duration | int | 10 | Benchmark duration in seconds | 
| -memory | bool | false | If true, uses an in-memory database; if false, uses a file database.db | 

## Database

The benchmark automatically creates a table and an index:

```sql
CREATE TABLE IF NOT EXISTS gastos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INT,
    valor REAL,
    data TEXT
);

CREATE INDEX IF NOT EXISTS idx_user_id ON gastos (user_id);
```

By default, the database is stored on disk as database.db.

For an in-memory benchmark, use:

```bash
go run main.go -memory=true
```
> Note: When using an in-memory database, the code ensures all threads share the same SQLite connection.

## Output

The benchmark prints to the terminal:

```bash
Writes per second: [....]
Reads per second: [....]
```
It also generates a PNG plot with the results:

```bash
benchmark_plot_<timestamp>.png
```

### Plot Details
- Blue line → Writes per second (Writes/s)
- Orange line → Reads per second (Reads/s)
- X-axis → Time in seconds
- Y-axis → Operations per second

## Example Usage
### Disk-based database
```bash
go run main.go -writers=4 -readers=8 -duration=15 -memory=false
```
### In-memory database
```bash
go run main.go -writers=2 -readers=10 -duration=10 -memory=true
```

## Notes

- The benchmark uses atomic counters to safely track reads and writes across goroutines.
- Reader threads execute queries limited to 10 rows to simulate lightweight reads.
- With WAL, SQLite allows reads to occur concurrently with writes.
- By default, go-sqlite3 sets a busy timeout of 5000ms (5s) and compiles SQLite with -DSQLITE_THREADSAFE=1.
- For in-memory databases with multiple connections, cache=shared ensures all goroutines share the same memory space and uses SetMaxOpenConns(1).
