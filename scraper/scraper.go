package scraper

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/qiniu/qmgo"
)

func ScrapeNews(maxPost, maxPaging int, client *qmgo.Client) error {
	baseURL := "https://nasional.sindonews.com/more/"

	for page := 5; page <= maxPaging; page++ {
		url := fmt.Sprintf("%s?page=%d", baseURL, page)
		doc, err := goquery.NewDocument(url)
		if err != nil {
			log.Printf("Failed to fetch web page: %v", err)
			continue
		}

		doc.Find("news-list li").Each(func(index int, item *goquery.Selection) {
			if index < maxPost {
				title := item.Find("h2").Text()
				author := item.Find("author").Text()
				dateStr := item.Find("date-kanal").Text()
				imageURL, _ := item.Find("detail-img").Attr("src")
				link, _ := item.Find("content-kanal-topik").Attr("href")

				// Split the date string to get the date part
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
				_, err := collection.InsertOne(context.TODO(), news)
				if err != nil {
					log.Printf("Failed to save to the database: %v", err)
				}
			}
		})
	}
	return nil
}
