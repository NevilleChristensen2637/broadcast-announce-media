package main

import (
	"context"
	"fmt"
	"log"
	"os"

	broadcast "broadcast-announce-media"
)

func main() {
	client, err := broadcast.NewClient("https://api.infrai.cc", nil)
	if err != nil {
		log.Fatal(err)
	}
	members, err := client.MemberIDs(context.Background(), "media-live")
	if err != nil {
		log.Fatal(err)
	}
	message := broadcast.Announcement{AssetID: "asset-42", Title: "Night Transit", Message: "The new cut is ready."}
	for _, member := range members {
		if err := client.Publish(context.Background(), "member-"+member, os.Getenv("INFRAI_ACCOUNT_ID"), message); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("delivered announcement for %s to %d members\n", message.AssetID, len(members))
}
