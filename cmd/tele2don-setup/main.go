package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"

	"github.com/mattn/go-mastodon"
)

func main() {
	flag.Parse()

	args := flag.Args()

	if len(args) != 1 {
		log.Fatalln("Invalid input.")
	}

	switch args[0] {
	case "mastodon":
		appCfg := &mastodon.AppConfig{
			ClientName: "tele2don",
			Scopes:     "read write",
			Website:    "https://github.com/merrkry/tele2don",
		}

		fmt.Println("Mastodon server address?")
		fmt.Scanln(&appCfg.Server)

		app, err := mastodon.RegisterApp(context.Background(), appCfg)
		if err != nil {
			log.Fatalln("Failed to register Mastodon app: ", err)
		}

		authUri, err := url.Parse(app.AuthURI)
		if err != nil {
			log.Fatalln("Failed to parse auth URI: ", err)
		}

		fmt.Printf("Please open the following URL in your browser to authorize the application: %s\n", authUri)
		fmt.Println("Authorization code?")
		var userAuthCode string
		fmt.Scanln(&userAuthCode)

		mastodonCfg := &mastodon.Config{
			Server:       appCfg.Server,
			ClientID:     app.ClientID,
			ClientSecret: app.ClientSecret,
		}
		client := mastodon.NewClient(mastodonCfg)
		err = client.GetUserAccessToken(context.Background(), userAuthCode, app.RedirectURI)
		if err != nil {
			log.Fatalln("Failed to create Mastodon client: ", err)
		}

		fmt.Println("Application registered successfully. Please load the following environment variables with tele2don.")
		fmt.Printf("MASTODON_SERVER=%s\n", appCfg.Server)
		fmt.Printf("MASTODON_CLIENT_ID=%s\n", app.ClientID)
		fmt.Printf("MASTODON_CLIENT_SECRET=%s\n", app.ClientSecret)
		fmt.Printf("MASTODON_ACCESS_TOKEN=%s\n", client.Config.AccessToken)
	default:
		log.Fatalln("Unsupported platform.")
	}
}
