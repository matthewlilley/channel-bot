package notifications

import (
	"github.com/BoltzExchange/channel-bot/lnd"
	"github.com/google/logger"
	"github.com/lightningnetwork/lnd/lnrpc"
)

func (manager *ChannelManager) checkClosedChannels() {
	logger.Info("Checking closed channels")

	closedChannels, err := manager.lnd.ClosedChannels()

	if err != nil {
		logger.Error("Could not get closed channels: " + err.Error())
		return
	}

	for _, channel := range closedChannels.Channels {
		// Do not send notifications for channels that were closed before the bot was started
		if channel.CloseHeight < manager.startupHeight {
			continue
		}

		// Do not send notifications for a closed channel more than once
		if sentAlready := manager.closedChannels[channel.ChanId]; sentAlready {
			continue
		}

		manager.logClosedChannel(channel)
		manager.closedChannels[channel.ChanId] = true
	}
}

func (manager *ChannelManager) logClosedChannel(channel *lnrpc.ChannelCloseSummary) {
	closeType := "closed"

	if channel.CloseType != lnrpc.ChannelCloseSummary_COOPERATIVE_CLOSE {
		closeType = "**force closed** :rotating_light:"
	}

	message := "Channel `" + lnd.FormatChannelID(channel.ChanId) + "` to `" + lnd.GetNodeName(manager.lnd, channel.RemotePubkey) + "` was " + closeType

	logger.Info(message)
	_ = manager.discord.SendMessage(message)
}
