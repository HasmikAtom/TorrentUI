package scraper

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func ScrapePirateBay(url string) ([]PirateBayTorrent, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var torrents []PirateBayTorrent

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),

		chromedp.WaitVisible(`main`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Extracting data...")
			return nil
		}),

		chromedp.Evaluate(`
			(() => {
				const entries = document.querySelectorAll('li.list-entry');
				const results = [];

				entries.forEach(entry => {
					const titleElement = entry.querySelector('.list-item.item-name a');
					const magnetElement = entry.querySelector('.item-icons a[href^="magnet"]');
					const uploadDateElement = entry.querySelector('.list-item.item-uploaded');
					const sizeElement = entry.querySelector('.list-item.item-size');
					const seedersElement = entry.querySelector('.list-item.item-seed');
					const leechersElement = entry.querySelector('.list-item.item-leech');
					const categoryElement = entry.querySelector('.list-item.item-type');
					const uploaderElement = entry.querySelector('.list-item.item-user');

					const torrent = {
						title: titleElement ? titleElement.textContent.trim() : '',
						magnet: magnetElement ? magnetElement.href : '',
						upload_date: uploadDateElement ? uploadDateElement.textContent.trim() : '',
						size: sizeElement ? sizeElement.textContent.trim() : '',
						se: seedersElement ? parseInt(seedersElement.textContent.trim(), 10) || 0 : 0,
						le: leechersElement ? parseInt(leechersElement.textContent.trim(), 10) || 0 : 0,
						category: categoryElement ? categoryElement.textContent.trim() : '',
						uploader: uploaderElement ? uploaderElement.textContent.trim() : '',
						torrent_link: titleElement ? titleElement.href : ''
					};

					results.push(torrent);
				});

				return results;
			})()
		`, &torrents),
	)

	if err != nil {
		log.Printf("Error: %v\n", err)
		return torrents, err
	}

	return torrents, nil
}
