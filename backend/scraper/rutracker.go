package scraper

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func ScrapeRuTracker(url string, torrentName string) ([]RutrackerTorrent, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	var results []RutrackerTorrent

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Extracting data...")
			return nil
		}),

		chromedp.WaitVisible(`a[onclick*="BB.toggle_top_login"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login button visible...")
			return nil
		}),

		chromedp.Click(`//b[contains(text(), "Вход")]/parent::a`, chromedp.BySearch),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login button clicked...")
			return nil
		}),

		chromedp.WaitEnabled(`#top-login-uname`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login fields available...")
			return nil
		}),
		chromedp.SendKeys(`#top-login-uname`, "HasmikAtom", chromedp.ByQuery),
		chromedp.SendKeys(`#top-login-pwd`, "57666777", chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login credentials filled...")
			return nil
		}),

		chromedp.Click(`#top-login-btn`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login button clicked...")
			return nil
		}),

		chromedp.Sleep(3*time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("waiting for the search bar and button...")
			return nil
		}),

		chromedp.WaitVisible(`#search-text`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("search button visible... searching...")
			return nil
		}),

		chromedp.SendKeys(`#search-text`, torrentName, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("search input filled...")
			return nil
		}),

		chromedp.Click(`#search-submit`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("search button clicked...")
			return nil
		}),

		chromedp.WaitVisible(`#tor-tbl`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("tor-table visible... evaluating....")
			return nil
		}),

		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('#tor-tbl tbody tr[id^="trs-tr-"]')).map(row => {
				const cells = row.cells;
				return {
					id: row.id.replace('trs-tr-', ''),
					category: cells[2]?.querySelector('.f-name a')?.textContent?.trim() || '',
					title: cells[3]?.querySelector('.t-title a')?.textContent?.trim() || '',
					author: cells[4]?.querySelector('.u-name a')?.textContent?.trim() || '',
					size: cells[5]?.querySelector('a')?.textContent?.trim() || '',
					downloadURL: cells[5]?.querySelector('a')?.href || '',
					se: cells[6]?.textContent?.trim() || '',
					le: cells[7]?.textContent?.trim() || '',
					downloads: cells[8]?.textContent?.trim() || '',
					dateAdded: cells[9]?.querySelector('p')?.textContent?.trim() || '',
				};
			})
		`, &results),
	)

	if err != nil {
		log.Printf("Error during initial scraping: %v\n", err)
		return results, err
	}

	return results, nil
}
