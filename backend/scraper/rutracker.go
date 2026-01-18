package scraper

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func ScrapeRuTracker(url string, torrentName string, creds RutrackerCredentials) ([]RutrackerTorrent, error) {
	return ScrapeRuTrackerWithTimeout(url, torrentName, creds, 120*time.Second)
}

func ScrapeRuTrackerWithTimeout(url string, torrentName string, creds RutrackerCredentials, timeout time.Duration) ([]RutrackerTorrent, error) {
	ctx, cancel := GetPool().NewTabContext(timeout)
	defer cancel()

	var results []RutrackerTorrent

	// Login first using shared auth
	if err := RutrackerLogin(ctx, url, creds); err != nil {
		log.Printf("Error during login: %v\n", err)
		return results, err
	}

	// Now perform the search
	err := chromedp.Run(ctx,
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
					title: cells[3]?.querySelector('.t-title a')?.textContent?.trim() || '',
					category: cells[2]?.querySelector('.f-name a')?.textContent?.trim() || '',
					uploader: cells[4]?.querySelector('.u-name a')?.textContent?.trim() || '',
					size: cells[5]?.querySelector('a')?.textContent?.trim() || '',
					upload_date: cells[9]?.querySelector('p')?.textContent?.trim() || '',
					se: cells[6]?.textContent?.trim() || '',
					le: cells[7]?.textContent?.trim() || '',
					description_url: cells[3]?.querySelector('.t-title a')?.href || '',

					download_url: cells[5]?.querySelector('a')?.href || '',

					downloads: cells[8]?.textContent?.trim() || '',
				};
			})
		`, &results),
	)

	if err != nil {
		log.Printf("Error during scraping: %v\n", err)
		return results, err
	}

	return results, nil
}
