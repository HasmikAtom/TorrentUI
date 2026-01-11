package scraper

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

type BrowserPool struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	mu          sync.Mutex
	initialized bool
}

var (
	pool     *BrowserPool
	poolOnce sync.Once
)

func GetPool() *BrowserPool {
	poolOnce.Do(func() {
		pool = &BrowserPool{}
	})
	return pool
}

func (p *BrowserPool) Init() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	p.allocCtx, p.allocCancel = chromedp.NewExecAllocator(context.Background(), opts...)
	p.initialized = true
	log.Println("Browser pool initialized")
	return nil
}

func (p *BrowserPool) NewTabContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	p.mu.Lock()
	if !p.initialized {
		p.mu.Unlock()
		p.Init()
		p.mu.Lock()
	}
	p.mu.Unlock()

	ctx, cancel := chromedp.NewContext(p.allocCtx)

	ctx, timeoutCancel := context.WithTimeout(ctx, timeout)

	return ctx, func() {
		timeoutCancel()
		cancel()
	}
}

func (p *BrowserPool) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.allocCancel != nil {
		p.allocCancel()
		p.initialized = false
		log.Println("Browser pool shut down")
	}
}

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
