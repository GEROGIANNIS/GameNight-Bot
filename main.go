package main

import (
	"fmt"
	"log"
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
}
