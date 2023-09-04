package scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/qiniu/qmgo"
)

func ScrapeNews(maxPost, maxPaging int, client *qmgo.Client) error {
	baseURL := "https://nasional.sindonews.com/more/%d"
	collection := client.Database("mynewsdb").Collection("news")

	var wg sync.WaitGroup
	errors := make(chan error, maxPost*maxPaging)

	scrapePage := func(page int) {
		defer wg.Done()
		url := fmt.Sprintf(baseURL, page)
		doc, err := goquery.NewDocument(url)
		if err != nil {
			errors <- fmt.Errorf("gagal mengambil halaman web: %v", err)
			return
		}

		if !hasData(doc) {
			return
		}

		doc.Find(".width-100.mb24.terkini").Each(func(index int, item *goquery.Selection) {
			title := item.Find(".desc-kanal.medium.width-100").Text()
			author := item.Find(".tipe-kanal.medium.sm-width-auto").Text()
			dateStr := item.Find(".date-kanal").Text()
			imageURL, _ := item.Find("img").Attr("data-src")
			link, _ := item.Find("a").Attr("href")

			title, author, dateStr, imageURL, content, err := scrapeArticleContent(link)
			if err != nil {
				log.Printf("Gagal mengambil data dari halaman artikel: %v", err)
				errors <- err
				return
			}

			dateParts := strings.Split(dateStr, " - ")
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

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err = collection.InsertOne(ctx, news)
			if err != nil {
				log.Printf("Gagal menyimpan ke database: %v", err)
				errors <- err
			}
		})
	}

	for page := 1; page <= maxPaging; page++ {
		wg.Add(1)
		go scrapePage(page)
	}

	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

func hasData(doc *goquery.Document) bool {
	return doc.Find(".width-100.mb24.terkini").Length() > 0
}

func scrapeArticleContent(link string) (string, string, string, string, string, error) {
	resp, err := http.Get(link)
	if err != nil {
		return "", "", "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", "", "", fmt.Errorf("Gagal mengambil halaman artikel: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", "", "", "", err
	}

	author := doc.Find(".detail-nama-redaksi a").Text()
	imageURL, _ := doc.Find(".detail-img img").Attr("data-src")
	title := doc.Find(".detail-title").Text()

	imageFileName := "images/" + getImageFileName(imageURL)

	resp, err = http.Get(imageURL)
	if err != nil {
		return "", "", "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", "", "", fmt.Errorf("Gagal mengambil gambar artikel: %s", resp.Status)
	}

	file, err := os.Create(imageFileName)
	if err != nil {
		return "", "", "", "", "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", "", "", "", "", err
	}

	var content strings.Builder
	doc.Find(".detail-desc").Each(func(index int, p *goquery.Selection) {
		text := p.Text()
		content.WriteString(text)
		content.WriteString("\n")
	})

	date := doc.Find(".detail-date-artikel").Text()

	return title, author, date, imageURL, content.String(), nil
}

func getImageFileName(imageURL string) string {
	parts := strings.Split(imageURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "image.jpg"
}
