package scraper

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

// RutrackerCredentials holds the login credentials
type RutrackerCredentials struct {
	Username string
	Password string
}

// RutrackerLogin performs login to RuTracker with provided credentials
func RutrackerLogin(ctx context.Context, url string, creds RutrackerCredentials) error {
	if creds.Username == "" || creds.Password == "" {
		log.Println("Warning: RuTracker credentials not configured")
	}

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Page loaded, looking for login button...")
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
		chromedp.SendKeys(`#top-login-uname`, creds.Username, chromedp.ByQuery),
		chromedp.SendKeys(`#top-login-pwd`, creds.Password, chromedp.ByQuery),

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
			log.Println("Waiting for login to complete...")
			return nil
		}),

		chromedp.WaitVisible(`#search-text`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Login successful, search bar visible...")
			return nil
		}),
	)

	if err != nil {
		log.Printf("Error during RuTracker login: %v", err)
		return err
	}

	return nil
}
