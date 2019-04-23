package main

import (
	"errors"
	"fmt"
	"github.com/mattn/go-shellwords"
	"github.com/nlopes/slack"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	Token      string
	CmdPattern string
	Users      []string
	Debug      bool
}

var config Config
var cmdPattern *regexp.Regexp
var users []string

func main() {
	// 設定ファイルの指定、読み取り
	exe, _ := os.Executable()
	configFile := filepath.Join(filepath.Dir(exe), "config.yml")

	rootCmd := &cobra.Command{
		Run: func(c *cobra.Command, args []string) {
			fmt.Printf("configFile: %s\n", configFile)
		},
	}
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", configFile, "config file path")
	cobra.OnInitialize(func() {
		viper.SetConfigFile(configFile)
		viper.AutomaticEnv()
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println("config file read error")
			fmt.Println(err)
			os.Exit(1)
		}
		if err := viper.Unmarshal(&config); err != nil {
			fmt.Println("config file Unmarshal error")
			fmt.Println(err)
			os.Exit(1)
		}
	})
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 受け付けるコマンドのパターンをコンパイル
	cmdPattern = regexp.MustCompile(config.CmdPattern)

	api := slack.New(config.Token)
	// slackでのUserIDの取得
	for _, email := range config.Users {
		user, err := api.GetUserByEmail(email)
		if err != nil {
			fmt.Sprintf("SlackID is not found: %s", email)
			continue
		}
		users = append(users, user.ID)
	}
	// Bot開始
	os.Exit(Run(api))
}

type SlackMessage struct {
	ChannelID string
	Options   []slack.MsgOption
}

func Run(api *slack.Client) int {
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	// Message送信用
	slackMessage := make(chan SlackMessage)
	defer close(slackMessage)
	go func(rtm *slack.RTM, slackMessage chan SlackMessage) {
		for {
			msg := <-slackMessage
			rtm.PostMessage(msg.ChannelID, msg.Options...)
		}
	}(rtm, slackMessage)

	for {
		select {
		case msg := <-rtm.IncomingEvents:

			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Bot開始
				log.Print("Start!")
			case *slack.InvalidAuthEvent:
				// 認証エラー
				log.Print("Invalid Credentials")
				return 1
			case *slack.MessageEvent:
				info := rtm.GetInfo()
				// 自身の投稿はスルー
				if ev.User == info.User.ID {
					continue
				}
				replayPattern := fmt.Sprintf("<@%s>", info.User.ID)
				// Bot宛てメッセージでなければスルー
				if !strings.Contains(ev.Msg.Text, replayPattern) {
					continue
				}

				// 新たなメッセージが投稿された際の挙動
				if config.Debug {
					log.Printf("Get Message: %s", ev.Msg.Text)
				}

				if err := ExecMessageEvent(rtm, ev, slackMessage); err != nil {
					log.Printf("Exec Error:%s", err.Error())
				}
			}
		}
	}
}

func ExecMessageEvent(rtm *slack.RTM, ev *slack.MessageEvent, slackMessage chan SlackMessage) error {
	// check user
	userOk := false
	for _, user := range users {
		if ev.User == user {
			userOk = true
			break
		}
	}
	if !userOk {
		msg := fmt.Sprintf("unknown user: %s", ev.User)
		rtm.SendMessage(rtm.NewOutgoingMessage(msg, ev.Channel))
		return errors.New(msg)
	}

	// リプライ部分を取り除く
	text := ev.Text
	rep := regexp.MustCompile(`<@\w+> `)
	text = rep.ReplaceAllString(text, "")

	// check command
	if !cmdPattern.Match([]byte(text)) {
		msg := fmt.Sprintf("unmatched pattern: %s", text)
		rtm.SendMessage(rtm.NewOutgoingMessage(msg, ev.Channel))
		return errors.New(msg)
	}

	log.Printf("Exec: %s", text)

	// gorutine
	go func(cmd string, channelID string, slackMessage chan SlackMessage) {
		attachment := slack.Attachment{
			Pretext: cmd,
		}
		c, err := shellwords.Parse(cmd)
		if err != nil {
			msg := fmt.Sprintf("command parse error: %s", err.Error())
			attachment.Color = "warning"
			attachment.Text = msg
			msgOpt := slack.MsgOptionAttachments(attachment)
			slackMessage <- SlackMessage{
				ChannelID: channelID,
				Options:   []slack.MsgOption{msgOpt},
			}

			return
		}
		var out []byte
		switch len(c) {
		case 0:
			return
		case 1:
			out, err = exec.Command(c[0]).CombinedOutput()
		default:
			out, err = exec.Command(c[0], c[1:]...).CombinedOutput()
		}
		if err != nil {
			msg := fmt.Sprintf("command exec error: %s", err.Error())
			attachment.Color = "danger"
			attachment.Text = msg
			msgOpt := slack.MsgOptionAttachments(attachment)
			slackMessage <- SlackMessage{
				ChannelID: channelID,
				Options:   []slack.MsgOption{msgOpt},
			}
			return
		}

		msg := string(out)
		attachment.Color = "good"
		attachment.Text = msg
		msgOpt := slack.MsgOptionAttachments(attachment)
		slackMessage <- SlackMessage{
			ChannelID: channelID,
			Options:   []slack.MsgOption{msgOpt},
		}
		return
	}(text, ev.Channel, slackMessage)
	return nil
}
