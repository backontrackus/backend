package main

import (
	"fmt"
	"log"
	"os"

	expo "github.com/oliveroneill/exponent-server-sdk-golang/sdk"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func announce(app *pocketbase.PocketBase, record *models.Record, token string) {
	client := expo.NewPushClient(nil)
	
	pushToken, err := expo.NewExponentPushToken(token)

	if err != nil {
		log.Default().Panicln(err)
	}

	app.Dao().ExpandRecord(record, []string{"user"}, nil)
	user := record.Expand()["user"].(*models.Record)
	user_name := user.GetString("name")
	title := record.GetString("title")
	body := fmt.Sprint("New announcement from ", user_name, ": ", title)

	response, err := client.Publish(
		&expo.PushMessage{
			To: []expo.ExponentPushToken{pushToken},
			Body: body,
			Title: title,
		},
	)

	if err != nil {
		log.Default().Panicln(err)
	}

	if response.ValidateResponse() != nil {
		fmt.Println(response.PushMessage.To, "failed")
	}
}

func main() {
	app := pocketbase.New()

	// serves static files from the provided public dir (if exists)
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))
		return nil
	})

	// Announcements notifications
	app.OnRecordAfterCreateRequest().Add(func(e *core.RecordCreateEvent) error {
		if e.Collection.Name == "announcements" {
			location := e.Record.GetString("location")

			if location == "" {
				devices, err := app.Dao().FindRecordsByFilter("devices", "token ~ \"ExponentPushToken\"", "-created", 0, 0, nil)

				if err != nil {
					log.Default().Panicln(err)
				}

				tokens := []string{}

				for _, v := range devices {
					tokens = append(tokens, v.GetString("token"))
				}

				for _, t := range tokens {
					announce(app, e.Record, t)
				}
			}

			users, err := app.Dao().FindRecordsByFilter("users", "location = {:location}", "-created", 0, 0, dbx.Params{"location": location})
		
			if err != nil {
				log.Default().Println(err)
			}

			tokens := []string{}

			for _, v := range users {
				app.Dao().ExpandRecord(v, []string{"devices"}, nil)

				devices := v.Expand()["devices"]

				if devices != nil {
					for _, v := range devices.([]*models.Record) {
						tokens = append(tokens, v.GetString("token"))
					}
				}
			}

			for _, t := range tokens {
				announce(app, e.Record, t)
			}
		}

		return nil
	})

	app.OnRecordAfterCreateRequest().Add(func (e *core.RecordCreateEvent) error {
		if e.Collection.Name == "messages" {
			app.Dao().ExpandRecord(e.Record, []string{"channel"}, nil)
			channel := e.Record.Expand()["channel"].(*models.Record)
			app.Dao().ExpandRecord(channel, []string{"users"}, nil)
			users := channel.Expand()["users"].([]*models.Record)
			
			tokens := []string{}

			for _, u := range users {
				if u.Id == e.Record.GetString("user") {
					continue
				}

				app.Dao().ExpandRecord(u, []string{"devices"}, nil)
				devices := u.Expand()["devices"]

				if devices != nil {
					for _, d := range devices.([]*models.Record) {
						tokens = append(tokens, d.GetString("token"))
					}
				}
			}

			client := expo.NewPushClient(nil)

			for _, t := range tokens {
				pushToken, err := expo.NewExponentPushToken(t)
			
				if err != nil {
					log.Default().Panicln(err)
				}
			
				app.Dao().ExpandRecord(e.Record, []string{"user"}, nil)
				user := e.Record.Expand()["user"].(*models.Record)
				user_name := user.GetString("name")
				title := fmt.Sprint("New announcement from ", user_name)
				body := e.Record.GetString("content")
			
				response, err := client.Publish(
					&expo.PushMessage{
						To: []expo.ExponentPushToken{pushToken},
						Body: body,
						Title: title,
					},
				)
			
				if err != nil {
					log.Default().Panicln(err)
				}
			
				if response.ValidateResponse() != nil {
					fmt.Println(response.PushMessage.To, "failed")
				}
			}
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
