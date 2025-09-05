package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

var Stop_flag = false
var Reads = int64(0)
var Writes = int64(0)
var Wg sync.WaitGroup

func writer_thread(db *sqlx.DB) {
	for {
		uid := rand.Intn(1000) + 1
		_, err := db.Exec("INSERT INTO gastos (user_id, valor, data) VALUES (?, ?, datetime('now'))", uid, rand.Float64()*100)
		if err != nil {
			fmt.Println("Error inserting:", err)
		}
		atomic.AddInt64(&Writes, 1)

		if Stop_flag {
			break
		}
	}
	Wg.Done()
}

func reader_thread(db *sqlx.DB) {
	for {
		uid := rand.Intn(1000) + 1
		result := db.QueryRowx("SELECT user_id FROM gastos WHERE user_id=? limit 10", uid)
		result.MapScan(map[string]interface{}{})

		atomic.AddInt64(&Reads, 1)

		if Stop_flag {
			break
		}
	}
	Wg.Done()
}

func main() {

	nWriters := flag.Int("writers", 2, "Número de threads de escrita")
	nReaders := flag.Int("readers", 10, "Número de threads de leitura")
	duration := flag.Int("duration", 10, "Duração do benchmark em segundos")
	useMemory := flag.Bool("memory", false, "Usar banco em memória")
	flag.Parse()

	dbFile := "database.db"
	if *useMemory {
		dbFile = "file:memdb1?mode=memory&cache=shared"
	}

	fmt.Printf("Iniciando benchmark com \n%d writers \n%d readers \n%d segundos \nem %s\n", *nWriters, *nReaders, *duration, dbFile)

	db, _ := sqlx.Open("sqlite3", dbFile)
	db.Exec("PRAGMA journal_mode=WAL")
	if *useMemory {
		db.SetMaxOpenConns(1)
	}
	db.Exec("CREATE TABLE IF NOT EXISTS gastos (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INT, valor REAL, data TEXT);")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_id ON gastos (user_id);")

	for i := 0; i < *nWriters; i++ {
		Wg.Add(1)
		go writer_thread(db)
	}

	for i := 0; i < *nReaders; i++ {
		Wg.Add(1)
		go reader_thread(db)
	}

	writes_slice := []int64{}
	reads_slice := []int64{}

	prevWrites := int64(0)
	prevReads := int64(0)
	for i := 0; i < *duration; i++ {
		// aguarda 1 segundo
		time.Sleep(1 * time.Second)
		writes_slice = append(writes_slice, Writes-prevWrites)
		reads_slice = append(reads_slice, Reads-prevReads)
		prevWrites = Writes
		prevReads = Reads
	}

	Stop_flag = true
	Wg.Wait()

	fmt.Println("Writes per second:")
	fmt.Println(writes_slice)

	fmt.Println("Reads per second:")
	fmt.Println(reads_slice)

	// Plotting the results
	plotBenchmark(writes_slice, reads_slice, *duration)

	defer db.Close()
}

func plotBenchmark(writes []int64, reads []int64, duration int) {
	p := plot.New()
	p.Title.Text = "SQLite WAL Benchmark"
	p.X.Label.Text = "Tempo (s)"
	p.Y.Label.Text = "Operações por segundo"

	ticks := make([]plot.Tick, duration)
	for i := 0; i < duration; i++ {
		ticks[i] = plot.Tick{Value: float64(i), Label: fmt.Sprintf("%ds", i)}
	}
	p.X.Tick.Marker = plot.ConstantTicks(ticks)

	// Create plot points for writes
	writesPoints := make(plotter.XYs, len(writes))
	for i, w := range writes {
		writesPoints[i].X = float64(i)
		writesPoints[i].Y = float64(w)
	}

	// Create plot points for reads
	readsPoints := make(plotter.XYs, len(reads))
	for i, r := range reads {
		readsPoints[i].X = float64(i)
		readsPoints[i].Y = float64(r)
	}

	p.Y.Tick.Marker = plot.TickerFunc(func(min, max float64) []plot.Tick {
		ticks := []plot.Tick{}
		step := (max - min) / 5 // 5 ticks
		for i := 0; i <= 5; i++ {
			val := min + float64(i)*step
			ticks = append(ticks, plot.Tick{Value: val, Label: fmt.Sprintf("%.0f", val)})
		}
		return ticks
	})

	// Add lines for writes and reads
	writesLine, _ := plotter.NewLine(writesPoints)
	writesLine.Color = plotutil.Color(0)
	readsLine, _ := plotter.NewLine(readsPoints)
	readsLine.Color = plotutil.Color(1)

	p.Add(writesLine, readsLine)
	p.Legend.Add("Writes/s", writesLine)
	p.Legend.Add("Reads/s", readsLine)
	p.Legend.Top = true

	// Save the plot to a PNG file
	if err := p.Save(10*vg.Inch, 5*vg.Inch, "benchmark_plot "+time.Now().Format(time.RFC3339)+".png"); err != nil {
		fmt.Println("Error saving plot:", err)
	}
}
