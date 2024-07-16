package main

import (
	"fmt"
	"log"
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
