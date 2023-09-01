package scraper

// News adalah struktur data untuk berita
type News struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	ImageURL string `json:"image_url"`
	Link     string `json:"link"`
}
