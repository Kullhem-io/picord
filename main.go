package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"
)

const (
	Token      = "#"
	ChannelID  = "1095198434903994368"
	RSSFeedURL = "https://rpilocator.com/feed/"
)

func main() {
	// Create a new Discord session
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	// Fetch the RSS feed
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(RSSFeedURL)
	if err != nil {
		fmt.Println("Error parsing RSS feed:", err)
		return
	}

	// Start a new goroutine to monitor the RSS feed
	go func() {
		for {
			// Check if any new items have been added to the RSS feed
			if len(feed.Items) > 0 {
				latestItem := feed.Items[0]
				if latestItem.PublishedParsed.After(time.Now().Add(-24 * time.Hour)) {
					// Notify the Discord channel about the new item
					message := fmt.Sprintf("%s is now in stock! %s", latestItem.Title, latestItem.Link)
					_, err = dg.ChannelMessageSend(ChannelID, message)
					if err != nil {
						fmt.Println("Error sending Discord message:", err)
					}
				}
			}

			// Wait for 5 minutes before checking the RSS feed again
			time.Sleep(5 * time.Minute)

			// Refresh the RSS feed
			resp, err := http.Get(RSSFeedURL)
			if err != nil {
				fmt.Println("Error fetching RSS feed:", err)
				continue
			}
			defer resp.Body.Close()
			feed, err = fp.Parse(resp.Body)
			if err != nil {
				fmt.Println("Error parsing RSS feed:", err)
			}
		}
	}()

	// Open a WebSocket connection to Discord and start listening for events
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session:", err)
		return
	}
	defer dg.Close()

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	<-make(chan struct{})
}
