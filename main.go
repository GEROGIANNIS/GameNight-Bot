package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("No token provided")
	}

	bot, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session,", err)
	}

	bot.AddHandler(messageCreate)

	err = bot.Open()
	if err != nil {
		log.Fatal("Error opening connection,", err)
	}

	fmt.Println("Bot is running. Press CTRL+C to exit.")

	// Start HTTP server
	go startHTTPServer()

	select {}
}

var serverTimezone *time.Location

func setTimezone(s *discordgo.Session, m *discordgo.MessageCreate, timezone string) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Invalid timezone: %s", timezone))
		return
	}
	serverTimezone = loc
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Timezone set to %s", timezone))
}

var games []string

func listGames(s *discordgo.Session, m *discordgo.MessageCreate, action, game string) {
	switch action {
	case "add":
		for _, g := range games {
			if g == game {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game %s is already in the list.", game))
				return
			}
		}
		games = append(games, game)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game %s added to the list", game))
	case "remove":
		for i, g := range games {
			if g == game {
				games = append(games[:i], games[i+1:]...)
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game %s removed from the list", game))
				return
			}
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game %s not found in the list", game))
	case "list":
		if len(games) == 0 {
			s.ChannelMessageSend(m.ChannelID, "No games in the list.")
		} else {
			gameList := "Games in the list:\n"
			for _, g := range games {
				gameList += fmt.Sprintf("- %s\n", g)
			}
			s.ChannelMessageSend(m.ChannelID, gameList)
		}
	default:
		s.ChannelMessageSend(m.ChannelID, "Invalid action. Use add/remove/list.")
	}
}

// New function to clear the games list
func clearGames(s *discordgo.Session, m *discordgo.MessageCreate) {
	games = []string{} // Reset the games slice to an empty slice
	s.ChannelMessageSend(m.ChannelID, "All games have been cleared from the list.")
}

func getCurrentTime(s *discordgo.Session, m *discordgo.MessageCreate) {
	if serverTimezone == nil {
		s.ChannelMessageSend(m.ChannelID, "Timezone not set. Use !set_timezone to set the timezone.")
		return
	}
	currentTime := time.Now().In(serverTimezone).Format("15:04:05 MST")
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Current time: %s", currentTime))
}

func getCurrentTimezone(s *discordgo.Session, m *discordgo.MessageCreate) {
	if serverTimezone == nil {
		s.ChannelMessageSend(m.ChannelID, "Timezone not set. Use !set_timezone to set the timezone.")
		return
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Current timezone: %s", serverTimezone.String()))
}

var announcementTime string

func setAnnouncementTime(s *discordgo.Session, m *discordgo.MessageCreate, time string) {
	announcementTime = time
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Announcement time set to %s", time))
}

func announceGame(s *discordgo.Session, channelID string) {
	rand.Seed(time.Now().UnixNano())
	if len(games) == 0 {
		s.ChannelMessageSend(channelID, "No games available to announce.")
		return
	}
	selectedGame := games[rand.Intn(len(games))]
	s.ChannelMessageSend(channelID, fmt.Sprintf("Tonight's game is %s! Click 'Join' if you have the game.", selectedGame))
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
		return
	}

	if m.Content == "!miaou" {
		s.ChannelMessageSend(m.ChannelID, "ti kaneis re malaka 💀")
		return
	}

	if strings.HasPrefix(m.Content, "!set_timezone ") {
		timezone := strings.TrimPrefix(m.Content, "!set_timezone ")
		timezone = strings.Trim(timezone, "\"")
		setTimezone(s, m, timezone)
		return
	}

	if m.Content == "!time" {
		getCurrentTime(s, m)
		return
	}

	if m.Content == "!timezone" {
		getCurrentTimezone(s, m)
		return
	}

	if strings.HasPrefix(m.Content, "!set_announcement_time ") {
		time := strings.TrimPrefix(m.Content, "!set_announcement_time ")
		setAnnouncementTime(s, m, time)
		return
	}

	if strings.HasPrefix(m.Content, "!clear_games") {
		clearGames(s, m)
		return
	}

	if strings.HasPrefix(m.Content, "!games ") {
		parts := strings.Fields(m.Content)
		if len(parts) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Usage: !games [add/remove/list] [game]")
			return
		}

		action := parts[1]

		// Allow the "list" action without requiring a game
		var game string
		if action != "list" {
			if len(parts) < 3 {
				s.ChannelMessageSend(m.ChannelID, "Usage: !games [add/remove/list] [game]")
				return
			}
			game = strings.Join(parts[2:], " ")
		}

		listGames(s, m, action, game)
		return
	}
}

func startHTTPServer() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		// Respond with the custom HTML content
		htmlContent := `<html><body><h1>GameNight Bot - GEROGIANNIS</h1></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))

		// Log the remote address that sent the ping request
		log.Printf("Received ping from: %s", r.RemoteAddr)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Get and print the server's IP addresses
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal("Error getting network interfaces:", err)
	}
	for _, address := range addresses {
		// Check if the address is not a loopback address and is an IP address
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				log.Printf("HTTP server will be accessible on: http://%s:%s/ping", ipNet.IP.String(), port)
			}
		}
	}

	log.Printf("HTTP server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Error starting HTTP server:", err)
	}
}
