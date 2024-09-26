package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	Games        []string `json:"games"`
	Timezone     string   `json:"timezone"`
	Announcement string   `json:"announcement"`
}

var configFileTemplate = "config_%s.json" // Template for config file names
var config Config
var serverTimezone *time.Location

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

	select {}
}

func loadConfig(serverID string) {
	configFile := fmt.Sprintf(configFileTemplate, serverID) // Create server-specific config file name
	file, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, initialize a new config
			config = Config{}
			saveConfig(serverID) // Create the file with empty values
			return
		}
		log.Fatal("Error opening config file:", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatal("Error decoding config file:", err)
	}
}

func saveConfig(serverID string) {
	configFile := fmt.Sprintf(configFileTemplate, serverID) // Create server-specific config file name
	file, err := os.Create(configFile)
	if err != nil {
		log.Fatal("Error creating config file:", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(config); err != nil {
		log.Fatal("Error encoding config file:", err)
	}
}

func setTimezone(s *discordgo.Session, m *discordgo.MessageCreate, timezone string) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Invalid timezone: %s", timezone))
		return
	}
	config.Timezone = timezone // Save timezone to config
	saveConfig(m.GuildID)      // Persist changes for this server
	serverTimezone = loc
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Timezone set to %s", timezone))
}

func listGames(s *discordgo.Session, m *discordgo.MessageCreate, action, game string) {
	switch action {
	case "add":
		for _, g := range config.Games {
			if g == game {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game %s is already in the list.", game))
				return
			}
		}
		config.Games = append(config.Games, game)
		saveConfig(m.GuildID) // Persist changes for this server
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game %s added to the list", game))
	case "remove":
		for i, g := range config.Games {
			if g == game {
				config.Games = append(config.Games[:i], config.Games[i+1:]...)
				saveConfig(m.GuildID) // Persist changes for this server
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game %s removed from the list", game))
				return
			}
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game %s not found in the list", game))
	case "list":
		if len(config.Games) == 0 {
			s.ChannelMessageSend(m.ChannelID, "No games in the list.")
		} else {
			gameList := "Games in the list:\n"
			for _, g := range config.Games {
				gameList += fmt.Sprintf("- %s\n", g)
			}
			s.ChannelMessageSend(m.ChannelID, gameList)
		}
	default:
		s.ChannelMessageSend(m.ChannelID, "Invalid action. Use add/remove/list.")
	}
}

func clearGames(s *discordgo.Session, m *discordgo.MessageCreate) {
	config.Games = []string{} // Reset the games slice to an empty slice
	saveConfig(m.GuildID)     // Persist changes for this server
	s.ChannelMessageSend(m.ChannelID, "All games have been cleared from the list.")
}

func getCurrentTime(s *discordgo.Session, m *discordgo.MessageCreate) {
	if config.Timezone == "" {
		s.ChannelMessageSend(m.ChannelID, "Timezone not set. Use !set_timezone to set the timezone.")
		return
	}
	loc, _ := time.LoadLocation(config.Timezone)
	currentTime := time.Now().In(loc).Format("15:04:05 MST")
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Current time: %s", currentTime))
}

func getCurrentTimezone(s *discordgo.Session, m *discordgo.MessageCreate) {
	if config.Timezone == "" {
		s.ChannelMessageSend(m.ChannelID, "Timezone not set. Use !set_timezone to set the timezone.")
		return
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Current timezone: %s", config.Timezone))
}

var announcementTime string

func setAnnouncementTime(s *discordgo.Session, m *discordgo.MessageCreate, time string) {
	announcementTime = time
	config.Announcement = time // Save announcement time to config
	saveConfig(m.GuildID)      // Persist changes for this server
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Announcement time set to %s", time))
}

func announceGame(s *discordgo.Session, channelID string) {
	rand.Seed(time.Now().UnixNano())
	if len(config.Games) == 0 {
		s.ChannelMessageSend(channelID, "No games available to announce.")
		return
	}
	selectedGame := config.Games[rand.Intn(len(config.Games))]
	s.ChannelMessageSend(channelID, fmt.Sprintf("Tonight's game is %s! Click 'Join' if you have the game.", selectedGame))
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Load the config specific to the server
	loadConfig(m.GuildID)

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
