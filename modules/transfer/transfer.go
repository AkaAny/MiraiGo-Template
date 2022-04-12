package transfer

import (
	"fmt"
	"github.com/Logiase/MiraiGo-Template/bot"
	"github.com/Logiase/MiraiGo-Template/config"
	"github.com/Logiase/MiraiGo-Template/utils"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"sync"
)

func init() {
	var instance = &transferModule{}
	bot.RegisterModule(instance)
}

type transferModule struct {
	tgBot    *tgbotapi.BotAPI
	tgChatID int64
	//msgCacheMap map[int64]message.IMessageElement
}

func (m *transferModule) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       "internal.transfer",
		Instance: m,
	}
}

func (m *transferModule) Init() {

}

func (m *transferModule) PostInit() {
	fmt.Println("transfer post init")
	var proxyAddress = config.GlobalConfig.GetString("tgbot.proxy.address")
	socks5ProxyDialer, err := proxy.SOCKS5("tcp", proxyAddress, nil, proxy.Direct)
	var transport = &http.Transport{Dial: func(network string, addr string) (net.Conn, error) {
		c, err := socks5ProxyDialer.Dial(network, addr)
		if err != nil {
			return nil, fmt.Errorf("proxy dial err:%w", err)
		}
		return c, nil
	}}
	var httpClient = &http.Client{Transport: transport}
	var tgBotToken = config.GlobalConfig.GetString("tgbot.token")
	tgbot, err := tgbotapi.NewBotAPIWithClient(tgBotToken, httpClient)
	if err != nil {
		panic(err)
	}
	tgbot.Debug = true
	var updateConfig = tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30
	updateChan, err := tgbot.GetUpdatesChan(updateConfig)
	if err != nil {
		panic(err)
	}
	var userName = config.GlobalConfig.GetString("tgbot.username")
	var expectedAuthStr = config.GlobalConfig.GetString("tgbot.authstr")
	go func() {
		for update := range updateChan {
			fmt.Println("user name:", update.Message.Chat.UserName)
			if update.Message.Chat.UserName != userName {
				continue
			}
			if !update.Message.IsCommand() {
				continue
			}
			var cmd = update.Message.Command()
			logger.WithField("tg cmd", cmd).Info()
			if cmd != "auth" {
				continue
			}
			var authStr = update.Message.CommandArguments()
			logger.WithField("tg auth str", authStr).Info()
			if authStr != expectedAuthStr {
				logger.Error("invalid auth str")
				continue
			}
			m.tgChatID = update.Message.Chat.ID
			_, err := tgbot.Send(tgbotapi.NewMessage(m.tgChatID, fmt.Sprintf("set chat id:%d", m.tgChatID)))
			if err != nil {
				logger.WithError(err).Error()
			}
		}
	}()
	fmt.Println("start receive tg chan msg")
	m.tgBot = tgbot
}

var logger = utils.GetModuleLogger("internal.transfer")

func (m *transferModule) Serve(b *bot.Bot) {
	b.OnGroupMessage(func(qqClient *client.QQClient, groupMessage *message.GroupMessage) {
		if m.tgChatID == 0 {
			return
		}
		var msgContent = groupMessage.ToString() + "\n"
		for _, elem := range groupMessage.Elements {
			switch elem.(type) {
			case *message.GroupImageElement:
				var groupImageElement = elem.(*message.GroupImageElement)
				msgContent += groupImageElement.Url
				break
			}
		}
		msgContent += "\n"
		var msgText = fmt.Sprintf("群聊：%s[%d]\n发送者：\n%s[%d]\n\n消息内容：%s",
			groupMessage.GroupName, groupMessage.GroupCode,
			groupMessage.Sender.DisplayName(), groupMessage.Sender.Uin,
			msgContent)
		var msgConfig = tgbotapi.NewMessage(m.tgChatID, msgText)
		_, err := m.tgBot.Send(msgConfig)
		if err != nil {
			logger.WithError(err).Error()
		}
	})

	b.OnPrivateMessage(func(qqClient *client.QQClient, privateMessage *message.PrivateMessage) {
		if m.tgChatID == 0 {
			return
		}
		var msgContent = privateMessage.ToString() + "\n"
		for _, elem := range privateMessage.Elements {
			switch elem.(type) {
			case *message.FriendImageElement:
				var friendImageElement = elem.(*message.FriendImageElement)
				msgContent += friendImageElement.Url
				break
			}
		}
		msgContent += "\n"
		var msgText = fmt.Sprintf("好友：%s[%d]\n\n\n消息内容：%s",
			privateMessage.Sender.DisplayName(), privateMessage.Sender.Uin,
			msgContent)
		var msgConfig = tgbotapi.NewMessage(m.tgChatID, msgText)
		_, err := m.tgBot.Send(msgConfig)
		if err != nil {
			logger.WithError(err).Error()
		}
	})

	b.OnTempMessage(func(qqClient *client.QQClient, event *client.TempMessageEvent) {

	})

	//b.OnDisconnected(func(qqClient *client.QQClient, event *client.ClientDisconnectedEvent) {
	//	logDisconnect(event)
	//})
}

func (m transferModule) Start(bot *bot.Bot) {

}

func (m transferModule) Stop(bot *bot.Bot, wg *sync.WaitGroup) {
	fmt.Println("transfer stop")
	m.tgBot.StopReceivingUpdates()
}
