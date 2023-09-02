package scraper

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/qiniu/qmgo"
)

func ScrapeNews(maxPost, maxPaging int, client *qmgo.Client) error {
	baseURL := "https://nasional.sindonews.com/more/"

	for page := 1; page <= maxPaging; page++ {
		url := fmt.Sprintf("%s%d", baseURL, page)
		doc, err := goquery.NewDocument(url)
		if err != nil {
			return fmt.Errorf("Gagal mengambil halaman web: %v", err)
		}

		doc.Find(".news").Each(func(index int, item *goquery.Selection) {
			if index < maxPost {
				title := item.Find(".news-title").Text()
				author := item.Find(".author").Text()
				dateStr := item.Find(".article-date").Text()
				imageURL, _ := item.Find(".news-image img").Attr("src")
				link, _ := item.Find(".news-title a").Attr("href")

				// Scraping isi artikel
				content, err := scrapeArticleContent(link)
				if err != nil {
					log.Printf("Gagal mengambil isi artikel: %v", err)
					// Lanjutkan dengan berita berikutnya jika gagal mengambil isi artikel.
					return
				}

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
					Content:  content,
				}

				// Simpan gambar sebagai file
				if imageURL != "" {
					err := saveImage(imageURL, title)
					if err != nil {
						log.Printf("Gagal menyimpan gambar: %v", err)
					}
				}

				// Simpan berita ke MongoDB
				collection := client.Database("mynewsdb").Collection("news")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				_, err = collection.InsertOne(ctx, news)
				if err != nil {
					log.Printf("Gagal menyimpan ke database: %v", err)
				}
			}
		})
	}
	return nil
}

func scrapeArticleContent(link string) (string, error) {
	resp, err := http.Get(link)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gagal mengambil halaman artikel: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	var content strings.Builder
	doc.Find(".article .article-paragraph p").Each(func(index int, p *goquery.Selection) {
		text := p.Text()
		content.WriteString(text)
		content.WriteString("\n")
	})

	return content.String(), nil
}

func saveImage(imageURL, title string) error {
	resp, err := http.Get(imageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Gagal mengambil gambar: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	imageFileName := fmt.Sprintf("%s.jpg", title)
	err = ioutil.WriteFile(imageFileName, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
