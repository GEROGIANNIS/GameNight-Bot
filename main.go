package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	ServerID          string   `json:"serverID"`
	Games             []string `json:"games"`
	Timezone          string   `json:"timezone"`
	Announcement      string   `json:"announcement"`  // Announcement time in "HH:MM" 24-hour format
	ParticipationList []string `json:"participation"` // List of users who confirmed participation
	VoiceChannelID    string   `json:"voice_channel"` // ID of the currently active voice channel
}

var configDir = "config"
var configFileTemplate = "config_%s.json"
var config Config
var updateAnnouncementCh = make(chan struct{}) // Channel to signal announcement time updates

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

	// Start the announcement checker in a separate goroutine
	go startAnnouncementScheduler(bot)

	select {} // Keep the program running
}

func loadConfig(serverID string) {
	configFile := filepath.Join(configDir, fmt.Sprintf(configFileTemplate, serverID))
	file, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			config = Config{}
			saveConfig(serverID)
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
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		log.Fatal("Error creating config directory:", err)
	}
	configFile := filepath.Join(configDir, fmt.Sprintf(configFileTemplate, serverID))
	config.ServerID = serverID
	file, err := os.Create(configFile)
	if err != nil {
		log.Fatal("Error creating config file:", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(config); err != nil {
		log.Fatal("Error encoding config file:", err)
	}
}

// Set the announcement time
func setAnnouncementTime(s *discordgo.Session, m *discordgo.MessageCreate, timeStr string) {
	_, err := time.Parse("15:04", timeStr) // Validate time format
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Invalid time format. Please use HH:MM (24-hour format).")
		return
	}

	config.Announcement = timeStr // Save announcement time to config
	saveConfig(m.GuildID)         // Persist changes for this server
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Announcement time set to %s", timeStr))

	// Notify all goroutines to update the announcement time
	updateAnnouncementCh <- struct{}{}
}

// Find the #todays-game channel in the guild and return its ID
func findTodaysGameChannel(s *discordgo.Session, guildID string) (string, error) {
	channels, err := s.GuildChannels(config.ServerID)
	if err != nil {
		return "", fmt.Errorf("error retrieving channels: %v", err)
	}

	for _, channel := range channels {
		if channel.Name == "todays-game" && channel.Type == discordgo.ChannelTypeGuildText {
			return channel.ID, nil
		}
	}

	return "", fmt.Errorf("todays-game channel not found")
}

// Create a voice channel with a dynamic name based on the game
func createVoiceChannel(s *discordgo.Session, guildID, gameName string) (string, error) {
	// Construct the channel name: "GameNight - <game_name>"
	channelName := fmt.Sprintf("GameNight - %s", gameName)

	// Prepare permission overwrites
	permissionOverwrites := []*discordgo.PermissionOverwrite{
		{
			ID:   guildID, // Use the guild ID for @everyone
			Type: discordgo.PermissionOverwriteTypeRole,
			Deny: discordgo.PermissionViewChannel, // Deny view and connect permissions
		},
	}

	for _, userID := range config.ParticipationList {
		permissionOverwrites = append(permissionOverwrites, &discordgo.PermissionOverwrite{
			ID:    userID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel, // Allow viewing and connecting
		})
	}

	// Create the channel with permission overwrites
	channel, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildVoice,
		PermissionOverwrites: permissionOverwrites,
	})
	if err != nil {
		return "", fmt.Errorf("error creating voice channel: %v", err)
	}
	return channel.ID, nil
}

// Update voice channel permissions based on the ParticipationList
func updateVoiceChannelPermissions(s *discordgo.Session, channelID string, guildID string) {
	permissionOverwrites := []*discordgo.PermissionOverwrite{
		{
			ID:   guildID, // Use the guild ID for @everyone
			Type: discordgo.PermissionOverwriteTypeRole,
			Deny: discordgo.PermissionViewChannel, // Deny view and connect permissions
		},
	}

	for _, userID := range config.ParticipationList {
		permissionOverwrites = append(permissionOverwrites, &discordgo.PermissionOverwrite{
			ID:    userID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel,
		})
	}

	// Update the channel's permission overwrites
	if _, err := s.ChannelEditComplex(channelID, &discordgo.ChannelEdit{
		PermissionOverwrites: permissionOverwrites,
	}); err != nil {
		log.Printf("Error updating voice channel permissions: %v", err)
	}
}

// Delete the voice channel
func deleteVoiceChannel(s *discordgo.Session, channelID string) {
	_, err := s.ChannelDelete(channelID)
	if err != nil {
		log.Printf("Error deleting voice channel: %v", err)
	}
}

// Announce a game in the #todays-game channel and create a voice channel
func announceGame(s *discordgo.Session, guildID string) {
	rand.Seed(time.Now().UnixNano())
	if len(config.Games) == 0 {
		log.Println("No games available to announce.")
		// Find the #todays-game channel and announce that there are no games available
		channelID, err := findTodaysGameChannel(s, config.ServerID)
		if err != nil {
			log.Println("Error finding #todays-game channel:", err)
			return
		}
		s.ChannelMessageSend(channelID, "No games available to announce today.")
		return
	}
	selectedGame := config.Games[rand.Intn(len(config.Games))]

	// Create a voice channel named "GameNight - <game_name>"
	voiceChannelID, err := createVoiceChannel(s, guildID, selectedGame)
	if err != nil {
		log.Printf("Error creating voice channel: %v", err)
		voiceChannelID = "" // If creation fails, continue with text announcement
	} else {
		config.VoiceChannelID = voiceChannelID
		saveConfig(guildID)
	}

	// Find the #todays-game channel and announce the game there
	channelID, err := findTodaysGameChannel(s, config.ServerID)
	if err != nil {
		log.Println("Error finding #todays-game channel:", err)
		return
	}

	// Announce the game and confirm participation
	message := fmt.Sprintf("Tonight's game is **%s**! Join the voice channel <#%s> to play! Use `!join` to confirm your participation.", selectedGame, voiceChannelID)
	s.ChannelMessageSend(channelID, message)

	// Schedule voice channel deletion after the game ends (e.g., 3 hours later)
	go func() {
		time.Sleep(15 * time.Second) // Adjust game duration as needed
		deleteVoiceChannel(s, voiceChannelID)
	}()
}

// Start the announcement scheduler
func startAnnouncementScheduler(s *discordgo.Session) {
	for {
		for _, guild := range s.State.Guilds {
			loadConfig(guild.ID) // Load the config for each guild

			if config.Announcement != "" && config.Timezone != "" {
				loc, err := time.LoadLocation(config.Timezone)
				if err != nil {
					log.Println("Error loading timezone:", err)
					time.Sleep(time.Minute)
					continue
				}

				now := time.Now().In(loc)
				announcementTime, err := time.ParseInLocation("15:04", config.Announcement, loc)
				if err != nil {
					log.Println("Error parsing announcement time:", err)
					time.Sleep(time.Minute)
					continue
				}

				// Adjust the announcement time to today
				announcementTime = time.Date(now.Year(), now.Month(), now.Day(),
					announcementTime.Hour(), announcementTime.Minute(), 0, 0, loc)

				// If the current time is past today's announcement time, schedule for tomorrow
				if now.After(announcementTime) {
					announcementTime = announcementTime.Add(24 * time.Hour)
				}

				// Log the calculated announcement time
				log.Printf("Next game announcement for guild %s scheduled at %s\n", guild.ID, announcementTime.Format("15:04"))

				// Wait for the duration until the next announcement or for an update
				for {
					durationUntilNextAnnouncement := time.Until(announcementTime)
					if durationUntilNextAnnouncement < 0 {
						// If the duration is negative, re-schedule for tomorrow
						announcementTime = announcementTime.Add(24 * time.Hour)
						durationUntilNextAnnouncement = time.Until(announcementTime)
					}
					log.Printf("Next game announcement for guild %s scheduled in %s\n", guild.ID, durationUntilNextAnnouncement)

					select {
					case <-time.After(durationUntilNextAnnouncement):
						announceGame(s, guild.ID) // Announce game dynamically using the guild ID
						break
					case <-updateAnnouncementCh:
						// Announcement time was updated, re-check the configuration
						log.Printf("Announcement time updated for guild %s, re-checking...\n", guild.ID)
						// Recalculate announcement time after update
						announcementTime, err = time.ParseInLocation("15:04", config.Announcement, loc)
						if err == nil {
							announcementTime = time.Date(now.Year(), now.Month(), now.Day(),
								announcementTime.Hour(), announcementTime.Minute(), 0, 0, loc)

							// If the new time is past now, no need to add a day
							if now.After(announcementTime) {
								announcementTime = announcementTime.Add(24 * time.Hour)
							}

							log.Printf("Next game announcement for guild %s recalculated to %s\n", guild.ID, announcementTime.Format("15:04"))
						} else {
							log.Println("Error re-parsing announcement time after update:", err)
						}

						break // Exit the inner loop to re-schedule announcements
					}
				}
			} else {
				log.Printf("No announcement time or timezone set for guild %s.\n", guild.ID)
			}
		}

		time.Sleep(time.Minute) // Recheck every minute
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Load the config specific to the server
	loadConfig(config.ServerID)

	if m.Content == "!help" {
		helpMessage := `
**Bot Commands:**
- !ping: Check if the bot is running.
- !set_timezone [timezone]: Set the server's timezone (e.g., "America/New_York").
- !set_announcement_time [HH:MM]: Set the time for daily announcements (24-hour format).
- !time: Get the current time in the set timezone.
- !timezone: Get the current timezone set for the server.
- !games [add/remove/list] [game]: Manage the list of games.
- !clear_games: Clear the list of games.
- !join: Confirm participation for the game.
- !leave: Remove participation from the game.
- !participants: List all confirmed participants.
- !help: Display this help message.
`
		s.ChannelMessageSend(m.ChannelID, helpMessage)
		return
	}

	if m.Content == "!ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
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
		timeStr := strings.TrimPrefix(m.Content, "!set_announcement_time ")
		setAnnouncementTime(s, m, timeStr)
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

	if m.Content == "!join" {
		joinParticipation(s, m, m.GuildID)
		return
	}

	// Add the new command for participants
	if m.Content == "!leave" {
		leaveParticipation(s, m, config.ServerID)
		return
	}

	if m.Content == "!participants" {
		listParticipants(s, m)
		return
	}
}

func setTimezone(s *discordgo.Session, m *discordgo.MessageCreate, timezone string) {
	// Validate timezone
	_, err := time.LoadLocation(timezone)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Invalid timezone. Please provide a valid timezone.")
		return
	}

	config.Timezone = timezone
	saveConfig(m.GuildID) // Persist changes for this server
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Timezone set to %s", timezone))

	// Notify all goroutines to update the announcement time
	updateAnnouncementCh <- struct{}{}
}

func getCurrentTime(s *discordgo.Session, m *discordgo.MessageCreate) {
	if config.Timezone == "" {
		s.ChannelMessageSend(m.ChannelID, "No timezone set. Use `!set_timezone [timezone]` to set it.")
		return
	}

	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error loading timezone.")
		return
	}

	now := time.Now().In(loc)
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Current time in %s: %s", config.Timezone, now.Format("15:04:05")))
}

func getCurrentTimezone(s *discordgo.Session, m *discordgo.MessageCreate) {
	if config.Timezone == "" {
		s.ChannelMessageSend(m.ChannelID, "No timezone set.")
		return
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Current timezone: %s", config.Timezone))
}

func clearGames(s *discordgo.Session, m *discordgo.MessageCreate) {
	config.Games = []string{}
	saveConfig(m.GuildID) // Persist changes for this server
	s.ChannelMessageSend(m.ChannelID, "Game list cleared.")
}

func listGames(s *discordgo.Session, m *discordgo.MessageCreate, action, game string) {
	switch action {
	case "add":
		config.Games = append(config.Games, game)
		saveConfig(m.GuildID) // Persist changes for this server
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game added: %s", game))

	case "remove":
		for i, g := range config.Games {
			if g == game {
				config.Games = append(config.Games[:i], config.Games[i+1:]...) // Remove game
				saveConfig(m.GuildID)                                          // Persist changes for this server
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Game removed: %s", game))
				return
			}
		}
		s.ChannelMessageSend(m.ChannelID, "Game not found in the list.")

	case "list":
		if len(config.Games) == 0 {
			s.ChannelMessageSend(m.ChannelID, "No games found.")
			return
		}

		gameList := "Games in the list:\n"
		for _, g := range config.Games {
			gameList += fmt.Sprintf("- %s\n", g)
		}
		s.ChannelMessageSend(m.ChannelID, gameList)

	default:
		s.ChannelMessageSend(m.ChannelID, "Invalid action. Use add/remove/list.")
	}
}

func joinParticipation(s *discordgo.Session, m *discordgo.MessageCreate, guildID string) {
	if config.ParticipationList == nil {
		config.ParticipationList = []string{}
	}

	// Check if the user is already in the participation list
	for _, participant := range config.ParticipationList {
		if participant == m.Author.ID {
			s.ChannelMessageSend(m.ChannelID, "You are already confirmed for participation.")
			return
		}
	}

	config.ParticipationList = append(config.ParticipationList, m.Author.ID) // Add user to participation
	saveConfig(m.GuildID)                                                    // Persist changes for this server
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s confirmed participation!", m.Author.Username))

	// Update voice channel permissions
	updateVoiceChannelPermissions(s, config.VoiceChannelID, guildID)
}

func leaveParticipation(s *discordgo.Session, m *discordgo.MessageCreate, guildID string) {
	if config.ParticipationList == nil {
		s.ChannelMessageSend(m.ChannelID, "No participants to leave from.")
		return
	}

	for i, participant := range config.ParticipationList {
		if participant == m.Author.ID {
			config.ParticipationList = append(config.ParticipationList[:i], config.ParticipationList[i+1:]...) // Remove user from participation
			saveConfig(m.GuildID)
			// Persist changes for this server
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s has left the participation list.", m.Author.Username))
			updateVoiceChannelPermissions(s, config.VoiceChannelID, guildID)
			return
		}
	}

	s.ChannelMessageSend(m.ChannelID, "You are not on the participation list.")
}

func listParticipants(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(config.ParticipationList) == 0 {
		s.ChannelMessageSend(m.ChannelID, "No participants yet.")
		return
	}

	// Create a list of usernames from user IDs
	var participants []string
	for _, participantID := range config.ParticipationList {
		user, err := s.User(participantID)
		if err == nil {
			participants = append(participants, user.Username)
		}
	}

	// Build the participant list message
	participantList := "Participants:\n"
	for _, username := range participants {
		participantList += fmt.Sprintf("- %s\n", username)
	}

	s.ChannelMessageSend(m.ChannelID, participantList)
}
