// main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adibhauzan/CLI-Scraper-Website/scraper"
	"github.com/qiniu/qmgo"
	"github.com/spf13/viper"
)

func main() {
	maxPost := flag.Int("max-post", 10, "Jumlah maksimum berita yang akan diambil")
	maxPaging := flag.Int("max-paging", 5, "Jumlah maksimum halaman yang akan di-scrape")

	flag.Parse()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Tidak dapat membaca file konfigurasi:", err)
		os.Exit(1)
	}

	mongoURL := viper.GetString("mongo_url")
	dbName := "mynewsdb"

	client, err := qmgo.NewClient(context.Background(), &qmgo.Config{Uri: mongoURL, Database: dbName})
	if err != nil {
		log.Fatalf("Gagal menghubungkan ke MongoDB: %v", err)
	}

	// Menutup koneksi ke MongoDB saat aplikasi selesai
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Timeout opsional
		defer cancel()

		if err := client.Close(ctx); err != nil {
			log.Fatalf("Gagal menutup koneksi ke MongoDB: %v", err)
		}
	}()

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
