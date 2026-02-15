package gateway

// Dispatcher is the interface used by HTTP handlers to dispatch events to
// connected WebSocket clients. The concrete Manager implements this interface.
type Dispatcher interface {
	DispatchToGuild(guildID int64, event string, data interface{})
	DispatchToUser(userID int64, event string, data interface{})
	DispatchToGuildExcept(guildID int64, exceptUserID int64, event string, data interface{})
	SubscribeToGuild(userID, guildID int64)
	UnsubscribeFromGuild(userID, guildID int64)
}
