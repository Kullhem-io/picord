package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
)

const (
	RSSFeedURL = "https://rpilocator.com/feed/"
)

func checkIfGUIDExists(db *sql.DB, guid string) (bool, error) {
	// prepare select statement
	stmt, err := db.Prepare("SELECT EXISTS (SELECT 1 FROM stock_alert WHERE guid = $1)")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	// execute select statement
	var exists bool
	err = stmt.QueryRow(guid).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func main() {

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	Token := os.Getenv("TOKEN")
	ChannelID := os.Getenv("CHANNELID")

	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	fmt.Println(psqlconn)
	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		fmt.Println("Database error", err)
		return
	}
	defer db.Close()

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

				for _, latestItem := range feed.Items {

					if latestItem.PublishedParsed.After(time.Now().Add(-24 * time.Hour)) {

						exists, err := checkIfGUIDExists(db, latestItem.GUID)
						if err != nil {
							panic(err)
						}

						if exists {
							continue
						}

						stmt, err := db.Prepare("INSERT INTO stock_alert (title, description, link, category1, category2, category3, guid, pubDate) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)")
						if err != nil {
							panic(err)
						}
						defer stmt.Close()

						pubDate, err := time.Parse(time.RFC1123, latestItem.Published)
						if err != nil {
							panic(err)
						}
						_, err = stmt.Exec(latestItem.Title,
							latestItem.Description,
							latestItem.Link,
							latestItem.Categories[0],
							latestItem.Categories[1],
							latestItem.Categories[2],
							latestItem.GUID,
							pq.FormatTimestamp(pubDate))
						if err != nil {
							panic(err)
						}

						// Notify the Discord channel about the new item
						message := fmt.Sprintf("%s in stock at %s -> %s", latestItem.Title, pubDate, latestItem.Link)
						_, err = dg.ChannelMessageSend(ChannelID, message)
						if err != nil {
							fmt.Println("Error sending Discord message:", err)
						}
					}
				}

			}

			// Wait for 30 seconds before checking the RSS feed again
			time.Sleep(30 * time.Second)

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
