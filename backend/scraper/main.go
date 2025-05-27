package scraper

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

type Scraper struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

func InitScraper() *Scraper {
	ctx, cancel := chromedp.NewContext(context.Background())

	return &Scraper{
		Ctx:    ctx,
		Cancel: cancel,
	}
}

func (s *Scraper) CheckLogin() {

}

func (s *Scraper) Login(url string) error {
	return chromedp.Run(s.Ctx,

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
	)

}
