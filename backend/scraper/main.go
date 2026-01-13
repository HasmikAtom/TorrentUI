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

// Scraper is deprecated - use RutrackerLogin from auth.go instead
type Scraper struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

// InitScraper is deprecated - use RutrackerLogin from auth.go instead
func InitScraper() *Scraper {
	ctx, cancel := chromedp.NewContext(context.Background())

	return &Scraper{
		Ctx:    ctx,
		Cancel: cancel,
	}
}
