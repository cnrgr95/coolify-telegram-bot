package main

//go:generate go run github.com/AshokShau/gotdbot/scripts/tools

import (
	"coolifymanager/src"
	"coolifymanager/src/config"
	"log"
	"strconv"
	"time"
	_ "time/tzdata"

	"github.com/AshokShau/gotdbot"
)

func main() {
	go startWebPanel()

	if err := config.InitConfig(); err != nil {
		log.Printf("failed to init config: %v", err)
		time.Sleep(10 * time.Minute)
		panic("failed to init config" + err.Error())
	}

	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Printf("⚠Failed to load Asia/Kolkata time zone: %v. Using UTC.", err)
	} else {
		time.Local = loc
	}

	apiID, err := strconv.Atoi(config.ApiId)
	if err != nil {
		log.Fatalf("❌ Invalid API_ID: %v", err)
	}

	tdlibLibraryPath := config.TdlibLibraryPath
	if tdlibLibraryPath == "" {
		tdlibLibraryPath = "./libtdjson.so.1.8.64"
	}

	bot, err := gotdbot.NewClient(int32(apiID), config.ApiHash, config.Token, &gotdbot.ClientOpts{
		LibraryPath: tdlibLibraryPath,
		AutoRetry:   &gotdbot.AutoRetry{MaxFloodWait: 5 * time.Minute, ChatNotFound: true},
	})

	if err != nil {
		log.Printf("❌ Failed to create bot client: %v", err)
		time.Sleep(10 * time.Minute)
		log.Fatalf("❌ Failed to create bot client: %v", err)
	}
	err = src.InitFunc(bot)
	if err != nil {
		log.Printf("failed to initialize bot: %v", err)
		time.Sleep(10 * time.Minute)
		panic("failed to initialize bot: " + err.Error())
	}

	if err = bot.Start(); err != nil {
		log.Printf("failed to start bot: %v", err)
		time.Sleep(10 * time.Minute)
		panic("failed to start bot: " + err.Error())
	}

	bot.Idle()
}

