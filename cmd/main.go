package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	// "time"

	"github.com/adibhauzan/CLI-Scraper-Website/internal/scraper"
	"github.com/qiniu/qmgo"
)

func main() {
	// Menggunakan konfigurasi eksternal
	var (
		maxPost   = flag.Int("max-post", 10, "Jumlah maksimum berita yang akan diambil")
		maxPaging = flag.Int("max-paging", 5, "Jumlah maksimum halaman yang akan di-scrape")
		mongoURL  = flag.String("mongo-url", "mongodb://localhost:27017", "MongoDB URI")
		dbName    = flag.String("db-name", "mynewsdb", "Nama database MongoDB")
	)

	flag.Parse()

	client, err := qmgo.NewClient(context.Background(), &qmgo.Config{Uri: *mongoURL, Database: *dbName})
	if err != nil {
		log.Fatalf("Gagal menghubungkan ke MongoDB: %v", err)
	}
	defer client.Close(context.Background())

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	done := make(chan bool)

	go func() {
		defer close(done)
		err := scraper.ScrapeNews(*maxPost, *maxPaging, client)
		if err != nil {
			log.Fatalf("Gagal melakukan scraping: %v", err)
		}
	}()

	select {
	case <-interrupt:
		fmt.Println("Menerima sinyal SIGINT (Ctrl+C). Menutup aplikasi...")
	case <-done:
		fmt.Println("Scraping selesai.")
	}
}
