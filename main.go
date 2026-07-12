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
	if err := config.InitConfig(); err != nil {
		log.Fatalf("failed to initialize configuration: %v", err)
	}

	go startWebPanel()

	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		log.Printf("Europe/Istanbul saat dilimi yüklenemedi: %v. UTC kullanılacak.", err)
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
		log.Fatalf("❌ Failed to create bot client: %v", err)
	}
	err = src.InitFunc(bot)
	if err != nil {
		log.Fatalf("failed to initialize bot: %v", err)
	}

	if err = bot.Start(); err != nil {
		log.Fatalf("failed to start bot: %v", err)
	}

	bot.Idle()
}
