package notifications

import (
	"github.com/BoltzExchange/channel-bot/discord"
	"github.com/BoltzExchange/channel-bot/lnd"
	"github.com/google/logger"
	"github.com/lightningnetwork/lnd/lnrpc"
	"strconv"
)

func (manager *ChannelManager) checkBalances() {
	logger.Info("Checking significant channel balances")

	channels, err := manager.lnd.ListChannels()

	if err != nil {
		logger.Error("Could not get channels: " + err.Error())
		return
	}

	manager.checkSignificantChannelBalances(channels.Channels)

	logger.Info("Checking normal channel balances")

	for _, channel := range channels.Channels {
		_, isSignificant := manager.significantChannels[channel.ChanId]

		if channel.UnsettledBalance != 0 || channel.Private || isSignificant {
			continue
		}

		channelRatio := getChannelRatio(channel)

		if channelRatio > 0.3 && channelRatio < 0.7 {
			if contains := manager.imbalancedChannels[channel.ChanId]; contains {
				manager.logBalance(channel, false)
				delete(manager.imbalancedChannels, channel.ChanId)
			}

			continue
		}

		// Do not send notifications for a balanced channel more than once
		if contains := manager.imbalancedChannels[channel.ChanId]; contains {
			continue
		}

		manager.imbalancedChannels[channel.ChanId] = true
		manager.logBalance(channel, true)
	}
}

func (manager *ChannelManager) checkSignificantChannelBalances(channels []*lnrpc.Channel) {
	for _, channel := range channels {
		significantChannel, isSignificant := manager.significantChannels[channel.ChanId]

		if channel.UnsettledBalance != 0 || !isSignificant {
			continue
		}

		channelRatio := getChannelRatio(channel)

		if channelRatio > significantChannel.ratios.max && channelRatio < significantChannel.ratios.min {
			if contains := manager.imbalancedChannels[channel.ChanId]; contains {
				significantChannel.logBalance(manager.discord, channel, false)
				delete(manager.imbalancedChannels, channel.ChanId)
			}

			continue
		}

		// Do not send notifications for an imbalanced significant channel more than once
		if contains := manager.imbalancedChannels[channel.ChanId]; contains {
			continue
		}

		manager.imbalancedChannels[channel.ChanId] = true
		significantChannel.logBalance(manager.discord, channel, true)
	}
}

func getChannelRatio(channel *lnrpc.Channel) float64 {
	return float64(channel.LocalBalance) / float64(channel.Capacity)
}

func (significantChannel *SignificantChannel) logBalance(discord discord.NotificationService, channel *lnrpc.Channel, isImbalanced bool) {
	var info string
	var emoji string

	if isImbalanced {
		info = "imbalanced"
		emoji = ":rotating_light:"
	} else {
		info = "balanced again"
		emoji = ":zap:"
	}

	message := emoji + " Channel " + significantChannel.Alias + " `" + lnd.FormatChannelID(channel.ChanId) + "` is **" + info + "** " + emoji + " :\n"

	localBalance, remoteBalance := formatChannelBalances(channel)
	message += localBalance + "\n"
	message += "    Minimal: " + formatFloat(float64(channel.Capacity)*significantChannel.ratios.min) + "\n"
	message += "    Maximal: " + formatFloat(float64(channel.Capacity)*significantChannel.ratios.max) + "\n"
	message += remoteBalance

	logger.Info(message)
	_ = discord.SendMessage(message)
}

func (manager *ChannelManager) logBalance(channel *lnrpc.Channel, isImbalanced bool) {
	var info string

	if isImbalanced {
		info = "imbalanced"
	} else {
		info = "balanced again"
	}

	message := "Channel `" + lnd.FormatChannelID(channel.ChanId) + "` to `" + lnd.GetNodeName(manager.lnd, channel.RemotePubkey) + "` is **" + info + "**:\n"

	localBalance, remoteBalance := formatChannelBalances(channel)
	message += localBalance + "\n" + remoteBalance

	logger.Info(message)
	_ = manager.discord.SendMessage(message)
}

func formatChannelBalances(channel *lnrpc.Channel) (local string, remote string) {
	local = "  Local: " + strconv.FormatInt(channel.LocalBalance, 10)
	remote = "  Remote: " + strconv.FormatInt(channel.RemoteBalance, 10)

	return local, remote
}

func formatFloat(float float64) string {
	return strconv.FormatFloat(float, 'f', 0, 64)
}
