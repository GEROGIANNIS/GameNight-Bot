
# GameNight Bot

GameNight Bot is designed to organize nightly game sessions within a Discord server. Each night, the bot announces a random game for the community to play, opening a dedicated voice channel for participants to join at midnight. This encourages regular gaming sessions, community bonding, and interactive fun.

## Commands

| Command                          | Description                                           |
|----------------------------------|-------------------------------------------------------|
| `!ping`                          | Check if the bot is running.                         |
| `!set_timezone [timezone]`      | Set the server's timezone (e.g., "America/New_York"). |
| `!set_announcement_time [HH:MM]`| Set the time for daily announcements (24-hour format). |
| `!time`                          | Get the current time in the set timezone.            |
| `!timezone`                      | Get the current timezone set for the server.         |
| `!games [add/remove/list] [game]`| Manage the list of games.                          |
| `!clear_games`                  | Clear the list of games.                             |
| `!join`                          | Confirm participation for the game.                  |
| `!leave`                         | Remove participation from the game.                  |
| `!participants`                  | List all confirmed participants.                     |


## Configuration

The bot will automatically create a configuration file in a `config` directory for each server it is invited to. The configuration includes:

- List of games
- Timezone for announcements
- Announcement time
- List of participants

## Acknowledgements

- [DiscordGo](https://github.com/bwmarrin/discordgo) - The library used to interact with the Discord API.

## Contributing

If you'd like to contribute to this project, feel free to open an issue or submit a pull request. All contributions are welcome!

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
