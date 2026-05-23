package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Command struct {
	Keywords []string `json:"keywords"`
	Replay   string   `json:"replay,omitempty"`
	Reaction string   `json:"reaction,omitempty"`
}

var Rules []Command

func loadCommands(filename string) error {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, &Rules)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	userMessage := strings.ToLower(m.Content)

	if strings.Contains(userMessage, "хуй") {
		emoji := []string{"%F0%9F%87%AD", "%f0%9f%87%ba", "%F0%9F%87%AE"}
		for _, e := range emoji {
			err := s.MessageReactionAdd(m.ChannelID, m.ID, e)
			if err != nil {
				fmt.Println("Error adding letter reaction: ", err)
			}
		}
		return
	}

	if strings.Contains(userMessage, "матвей") {
		imagePath := "matvey.png"

		file, err := os.Open(imagePath)
		if err != nil {
			fmt.Println("Error opening image for Matvey: ", err)
			s.ChannelMessageSend(m.ChannelID, "Не удалось найти картинку для Матвея!")
			return
		}
		defer file.Close()

		msgData := &discordgo.MessageSend{
			Content: "На месте епт",
			Files: []*discordgo.File{
				{
					Name:        "matvey.png",
					ContentType: "image/png",
					Reader:      file,
				},
			},
		}

		_, err = s.ChannelMessageSendComplex(m.ChannelID, msgData)
		if err != nil {
			fmt.Println("Error sending image for Matvey: ", err)
		}
		return
	}

	for _, rule := range Rules {
		for _, keyword := range rule.Keywords {
			if strings.Contains(userMessage, strings.ToLower(keyword)) {
				if rule.Replay != "" {
					s.ChannelMessageSend(m.ChannelID, rule.Replay)
				}
				if rule.Reaction != "" {
					err := s.MessageReactionAdd(m.ChannelID, m.ID, rule.Reaction)
					if err != nil {
						fmt.Println("Error adding reaction: ", err)
					}
				}
				return
			}
		}
	}
}

func main() {
	TokenBot := os.Getenv("DISCORD_TOKEN")
	if TokenBot == "" {
		panic("Error: bot token not initialization")
	}

	err := loadCommands("command.json")
	if err != nil {
		fmt.Println("Error load json: ", err)
		return
	}
	fmt.Println("json loaded successfully")

	bt, err := discordgo.New("Bot " + TokenBot)
	if err != nil {
		fmt.Println("Error creating discord session: ", err)
		return
	}

	bt.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions
	bt.AddHandler(messageCreate)

	err = bt.Open()
	if err != nil {
		fmt.Println("Error opening connection: ", err)
		return
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	bt.Close()
}
