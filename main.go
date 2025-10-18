package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/krognol/go-wolfram"
	"github.com/shomali11/slacker"
	"github.com/tidwall/gjson"

	witai "github.com/wit-ai/wit-go/v2"
)

var wolframClient *wolfram.Client

func printCommandEvents(analyticsChannel <-chan *slacker.CommandEvent) {
	for event := range analyticsChannel {
		fmt.Println("Command Events")
		fmt.Println(event.Timestamp)
		fmt.Println(event.Command)
		fmt.Println(event.Parameters)
		fmt.Println(event.Event)
		fmt.Println()
	}
}

func main() {
	godotenv.Load(".env")

	bot := slacker.NewClient(os.Getenv("SLACK_BOT_TOKEN"), os.Getenv("SLACK_APP_TOKEN")) // to get access token use https://api.slack.com/custom-integrations/legacy-tokens
	client := witai.NewClient(os.Getenv("WIT_AI_TOKEN"))                                 // to get access token use https://wit.ai
	wolframClient = &wolfram.Client{AppID: os.Getenv("WOLFRAM_APP_ID")}                  // to get access token use https://www.wolframalpha.com/
	go printCommandEvents(bot.CommandEvents())                                           // print function from slacker package to get command events

	bot.Command("query for bot - <message>", &slacker.CommandDefinition{
		Description: "send any question to wolfram",
		Example:     "who is the president of india",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			query := request.Param("message") // user to slack message

			msg, err := client.Parse(&witai.MessageRequest{ // slack to wit.ai message
				Query: query,
			})
			if err != nil {
				log.Printf("error calling Wit.ai: %v", err)
				response.Reply("Sorry, I'm having trouble understanding right now.")
				return
			}

			data, err := json.MarshalIndent(msg, "", "    ") // convert to json
			if err != nil {
				log.Printf("error marshalling wit.ai response: %v", err)
				response.Reply("Sorry, I'm having trouble processing the response.")
				return
			}

			rough := string(data[:])
			value := gjson.Get(rough, "entities.wit$wolfram_search_query:wolfram_search_query.0.value")
			answer := query // Fallback to the original query
			if value.Exists() {
				answer = value.String()
			}

			res, err := wolframClient.GetSpokentAnswerQuery(answer, wolfram.Metric, 1000)
			if err != nil {
				log.Printf("wolfram query failed: %v", err)
				response.Reply("Sorry, I couldn't get an answer from Wolfram Alpha.")
				return
			}
			response.Reply(res)
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := bot.Listen(ctx)

	if err != nil {
		log.Fatal(err)
	}
}
