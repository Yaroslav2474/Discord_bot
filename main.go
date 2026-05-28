package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Structura struct {
	First string `json:"message"`
}

type Command struct {
	Keywords []string `json:"keywords"`
	Replay   string   `json:"replay,omitempty"`
	Reaction string   `json:"reaction,omitempty"`
}

var Rules []Command

func request_to_API() (string, error) {
	url := "https://dog.ceo/api/breeds/image/random"

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("Error when sending the get request: ", err)
		return "", fmt.Errorf("1")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("Error when read: ", err)
		return "", fmt.Errorf("1")
	}

	var hz Structura

	err = json.Unmarshal(body, &hz)

	if err != nil {
		fmt.Println("Error when unmarshaling: ", err)
		return "", fmt.Errorf("1")
	}

	img := hz.First

	return img, nil

}

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

	if strings.Contains(userMessage, "funy time") || strings.Contains(userMessage, "поднять настроение") || strings.Contains(userMessage, "1") {
		url_img, err := request_to_API()

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to execute the function :(")
			return
		}

		s.ChannelMessageSend(m.ChannelID, url_img)
		return
	}

	if strings.Contains(userMessage, "андреич") {
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
