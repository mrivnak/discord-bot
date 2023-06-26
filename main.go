package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type Role struct {
	Name  string `json:"roleName"`
	Emoji string `json:"roleEmote"`
}
type RoleMessage struct {
	MessageID string `json:"messageId"`
	Roles     []Role `json:"roles"`
}

type RolesConfig struct {
	Items []RoleMessage `json:"items"`
}

type Config struct {
	RolesChannelID string `json:"rolesChannelId"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		os.Exit(1)
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		fmt.Println("No token provided. Please set DISCORD_TOKEN environment variable.")
		os.Exit(1)
	}

	// Create a new Discord session using the provided bot token.
	disc, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		os.Exit(1)
	}

	disc.AddHandler(createRoleMessages)

	disc.AddHandler(compHandler)
	disc.AddHandler(rolesAddHandler)
	disc.AddHandler(rolesRemoveHandler)

	disc.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentGuildMembers)

	// Open a websocket connection to Discord and begin listening.
	err = disc.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	disc.Close()
}

func compHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// TODO: use regex for more flexibility
	if m.Content == "comp?" {
		s.ChannelMessageSend(m.ChannelID, "comp? :eyes: @CSGO")
		fmt.Println("comp requested")
	}

	if m.Content == "no comp" {
		s.ChannelMessageSend(m.ChannelID, "Damn :(")
		// TODO: clear queue
	}

	// TODO: add a timeout for the queue

	// TODO: notify when queue is full
	// TODO: notify if queue is over capacity
	// TODO: allow user to give up their spot
	// TODO: request to join waitlist and notify when a spot opens up
}

func createRoleMessages(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == ";create-role-messages" {
		config, err := readFromJson[Config]("config/config/config.json")
		if err != nil {
			fmt.Println("error reading config,", err)
			return
		} else {
			fmt.Println("config loaded")
		}
		rolesConfig, err := readFromJson[RolesConfig]("config/roles.json")
		if err != nil {
			fmt.Println("error reading roles config,", err)
			return
		} else {
			fmt.Println("roles config loaded")
		}

		fmt.Println("creating role messages in channel " + config.RolesChannelID)

		for i := 0; i < len(rolesConfig.Items); i++ {
			roleMessage := &rolesConfig.Items[i]
			fmt.Println("creating role message for message " + roleMessage.MessageID)

			messageText := "React to choose a role:\n"
			for _, role := range roleMessage.Roles {
				messageText += role.Emoji + " - " + role.Name + "\n"
			}
			message, err := s.ChannelMessageSend(string(config.RolesChannelID), messageText)
			if err != nil {
				fmt.Println("error creating role message,", err)
				return
			} else {
				roleMessage.MessageID = message.ID
			}
		}

		err = saveJsonToFile("config/roles.json", rolesConfig)
		if err != nil {
			fmt.Println("error saving roles config,", err)
			return
		} else {
			fmt.Println("roles config saved")
		}

		// TODO: add reactions to messages
		for _, roleMessage := range rolesConfig.Items {
			for _, role := range roleMessage.Roles {
				fmt.Println("adding reaction " + role.Emoji + " to message " + roleMessage.MessageID)
				err := s.MessageReactionAdd(config.RolesChannelID, roleMessage.MessageID, role.Emoji)
				if err != nil {
					fmt.Println("error adding reaction,", err)
					return
				}
			}
		}
	}
}

func rolesAddHandler(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if m.UserID == s.State.User.ID {
		return
	}

	fmt.Println("reaction added")

	config, err := readFromJson[Config]("config/config.json")
	if err != nil {
		fmt.Println("error reading config,", err)
		return
	}
	rolesConfig, err := readFromJson[RolesConfig]("config/roles.json")
	if err != nil {
		fmt.Println("error reading roles config,", err)
		return
	}

	if m.ChannelID != config.RolesChannelID {
		return
	}

	for _, roleMessage := range rolesConfig.Items {
		if m.MessageID == roleMessage.MessageID {
			for _, role := range roleMessage.Roles {
				if m.Emoji.APIName() == role.Emoji {
					fmt.Println("adding role " + role.Name + " to user " + m.UserID)

					allRoles, err := s.GuildRoles(m.GuildID)
					if err != nil {
						fmt.Println("error getting guild roles,", err)
						return
					}

					for _, r := range allRoles {
						if r.Name == role.Name {
							err = s.GuildMemberRoleAdd(m.GuildID, m.UserID, r.ID)
							if err != nil {
								fmt.Println("error adding role to user,", err)
								return
							}
						}
					}

				}
			}
		}
	}
}

func rolesRemoveHandler(s *discordgo.Session, m *discordgo.MessageReactionRemove) {
	if m.UserID == s.State.User.ID {
		return
	}

	fmt.Println("reaction removed")

	config, err := readFromJson[Config]("config/config.json")
	if err != nil {
		fmt.Println("error reading config,", err)
		return
	}
	rolesConfig, err := readFromJson[RolesConfig]("config/roles.json")
	if err != nil {
		fmt.Println("error reading roles config,", err)
		return
	}

	if m.ChannelID != config.RolesChannelID {
		return
	}

	for _, roleMessage := range rolesConfig.Items {
		if m.MessageID == roleMessage.MessageID {
			for _, role := range roleMessage.Roles {
				if m.Emoji.APIName() == role.Emoji {
					fmt.Println("removing role " + role.Name + " from user " + m.UserID)

					allRoles, err := s.GuildRoles(m.GuildID)
					if err != nil {
						fmt.Println("error getting guild roles,", err)
						return
					}

					for _, r := range allRoles {
						if r.Name == role.Name {
							err = s.GuildMemberRoleRemove(m.GuildID, m.UserID, r.ID)
							if err != nil {
								fmt.Println("error removing role from user,", err)
								return
							}
						}
					}

				}
			}
		}
	}
}

func readFromJson[T any](filename string) (T, error) {
	fileContent, err := os.Open(filename)
	if err != nil {
		var result T
		return result, err
	}

	defer fileContent.Close()

	byteResult, err := ioutil.ReadAll(fileContent)
	if err != nil {
		var result T
		return result, err
	}

	var result T
	err = json.Unmarshal(byteResult, &result)

	return result, err
}

func saveJsonToFile[T any](filename string, data T) error {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}
