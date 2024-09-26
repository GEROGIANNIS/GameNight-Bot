GameNight Bot

GameNight Bot is designed to organize nightly game sessions within a Discord server.
Each night, the bot announces a random game for the community to play, opening a dedicated voice channel for participants to join at midnight.
This encourages regular gaming sessions, community bonding, and interactive fun.

### Commands:

- `!ping`: Test if the bot is responding.
- `!miaou`: Get a fun response from the bot.
- `!set_timezone [timezone]`: Set the timezone for the server (e.g., `!set_timezone America/New_York`).
- `!time`: Get the current time based on the server's timezone.
- `!timezone`: Display the current timezone set for the server.
- `!set_announcement_time [time]`: Set a time for announcements (e.g., `!set_announcement_time 18:00`).
- `!games [add/remove/list] [game]`: Manage the list of games.
  - Example: `!games add Chess`, `!games remove Monopoly`, `!games list`.
- `!clear_games`: Remove all games from the list.

## Configuration

The bot creates a separate configuration file for each server it is added to, named `config_<serverID>.json`. This file contains:

- **Games**: List of games for that server.
- **Timezone**: The server's timezone setting.
- **Announcement Time**: The time set for game night announcements.

## Contributing

If you'd like to contribute to this project, feel free to open an issue or submit a pull request. All contributions are welcome!

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgements

- [DiscordGo](https://github.com/bwmarrin/discordgo) - The library used to interact with the Discord API.
