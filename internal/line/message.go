package line

import (
	"github.com/line/line-bot-sdk-go/v8/linebot"
)

func PushMessage(bot *linebot.Client, userID string, message string) error {
	_, err := bot.PushMessage(userID, linebot.NewTextMessage(message)).Do()
	if err != nil {
		return err
	}
	return nil
}
