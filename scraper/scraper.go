package scraper

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/qiniu/qmgo"
)

func SaveImage(url, filePath string) error {
	// Mengambil gambar dari URL
	respons, err := http.Get(url)
	if err != nil {
		return err
	}
	defer respons.Body.Close()

	// Membuat file keluaran
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Menyalin data gambar ke file keluaran
	_, err = io.Copy(file, respons.Body)
	if err != nil {
		return err
	}

	return nil
}

func ScrapeNews(maxPost, maxPaging *int, client *qmgo.Client) error {
	for page := 1; page <= *maxPaging; page++ {
		url := fmt.Sprintf("https://nasional.sindonews.com/more/%d", page)
		doc, err := goquery.NewDocument(url)
		if err != nil {
			log.Printf("Gagal mengakses halaman %d: %v", page, err)
			continue
		}

		doc.Find(".news-list li").Each(func(index int, item *goquery.Selection) {
			if index < *maxPost {
				title := item.Find("h2").Text()
				author := item.Find(".author").Text()
				dateStr := item.Find(".news-date").Text()
				imageURL, _ := item.Find("img").Attr("src")
				link, _ := item.Find("a").Attr("href")

				dateParts := strings.Split(dateStr, ", ")
				if len(dateParts) == 2 {
					dateStr = dateParts[1]
				}

				news := News{
					Title:    title,
					Author:   author,
					Date:     dateStr,
					ImageURL: imageURL,
					Link:     link,
				}

				collection := client.Database("mynewsdb").Collection("news")
				_, err = collection.InsertOne(nil, news)
				if err != nil {
					log.Printf("Gagal menyimpan ke database: %v", err)
				}

				err = SaveImage(imageURL, fmt.Sprintf("images/%d.jpg", index))
				if err != nil {
					log.Printf("Gagal menyimpan gambar: %v", err)
				}
			}
		})
	}

	return nil
}
