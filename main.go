package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

var gmaps = "https://www.google.com/maps/search/%s"

type Place struct {
	Name    string
	Phone   string
	Address string
}

var banner = "Hypnos by github.com/alf4ridzi"

func main() {
	fmt.Println()
	fmt.Println(banner)
	fmt.Println()

	var keyword string
	fmt.Print("Keyword : ")
	fmt.Scan(&keyword)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.WindowSize(1200, 800),
	)

	alloCtx, cancelAllo := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAllo()

	taskCtx, cancelTask := chromedp.NewContext(alloCtx, chromedp.WithLogf(log.Printf))
	defer cancelTask()

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, 600*time.Second)
	defer cancelTimeout()

	err := chromedp.Run(taskCtx,
		chromedp.Navigate(fmt.Sprintf(gmaps, keyword)),
		chromedp.Sleep(3*time.Second),
		chromedp.WaitVisible(`div[role="feed"]`, chromedp.ByQuery),
	)
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Create(keyword + ".csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"Name", "Phone", "Address"})

	visited := map[string]bool{}
	places := []Place{}

	for {
		var hrefs []string
		err = chromedp.Run(taskCtx,
			chromedp.Evaluate(`
				Array.from(document.querySelectorAll('div[role="feed"] a[href*="/maps/place/"]'))
					.map(el => el.href)
			`, &hrefs),
		)
		if err != nil {
			log.Println(err)
			break
		}

		newFound := false
		for _, href := range hrefs {
			if visited[href] {
				continue
			}
			newFound = true
			visited[href] = true

			var name, phone, address string

			err = chromedp.Run(taskCtx,
				chromedp.Evaluate(fmt.Sprintf(`
					var items = document.querySelectorAll('div[role="feed"] a[href*="/maps/place/"]');
					for (var i = 0; i < items.length; i++) {
						if (items[i].href === %q) {
							items[i].scrollIntoView({behavior: 'smooth', block: 'center'});
							items[i].click();
							break;
						}
					}
				`, href), nil),
				chromedp.Sleep(3*time.Second),
				chromedp.Evaluate(`
					var el = document.querySelector('h1.DUwDvf');
					el ? el.innerText.trim() : ''
				`, &name),
				chromedp.Evaluate(`
					(function() {
						var buttons = document.querySelectorAll('button');
						for (var i = 0; i < buttons.length; i++) {
							var di = buttons[i].getAttribute('data-item-id') || '';
							if (di.includes('phone')) {
								var span = buttons[i].querySelector('[class*="Io6YTe"]')
									|| buttons[i].querySelector('div > div > div');
								return span ? span.innerText.trim() : buttons[i].innerText.trim();
							}
						}
						return '';
					})()
				`, &phone),
				chromedp.Evaluate(`
					(function() {
						var buttons = document.querySelectorAll('button');
						for (var i = 0; i < buttons.length; i++) {
							var di = buttons[i].getAttribute('data-item-id') || '';
							if (di === 'address') {
								var span = buttons[i].querySelector('[class*="Io6YTe"]')
									|| buttons[i].querySelector('div > div > div');
								return span ? span.innerText.trim() : buttons[i].innerText.trim();
							}
						}
						return '';
					})()
				`, &address),
			)
			if err != nil {
				log.Println("error:", err)
				continue
			}

			p := Place{Name: name, Phone: phone, Address: address}
			places = append(places, p)
			writer.Write([]string{name, phone, address})
			writer.Flush()

			fmt.Printf("[%d] %s | %s | %s\n", len(places), name, phone, address)
		}

		var reachedEnd bool
		chromedp.Run(taskCtx,
			chromedp.Evaluate(`
				var spans = document.querySelectorAll('div[role="feed"] span');
				var reached = false;
				for (var i = 0; i < spans.length; i++) {
					if (spans[i].innerText && spans[i].innerText.includes("You've reached")) {
						reached = true;
						break;
					}
				}
				reached;
			`, &reachedEnd),
		)

		if reachedEnd && !newFound {
			fmt.Println("Done.")
			break
		}

		chromedp.Run(taskCtx,
			chromedp.Evaluate(`
				var feed = document.querySelector('div[role="feed"]');
				if (feed) feed.scrollTop += 800;
			`, nil),
			chromedp.Sleep(2*time.Second),
		)
	}

	fmt.Printf("Saved to %s.csv\n", len(places), keyword)
}

func scrollFeed(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.Evaluate(`
			var feed = document.querySelector('div[role="feed"]');
			if (feed) feed.scrollTop += 800;
		`, nil),
		chromedp.Sleep(2*time.Second),
	)
}
