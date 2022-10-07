package wire

import "time"

type Protocol string

type Magic [4]byte

var (
	MagicLogicPkt = Magic{0xc3, 0x11, 0xa3, 0x65}
	MagicBasicPkt = Magic{0xc3, 0x15, 0xa7, 0x65}
)

const (
	AlgorithmHashSlots = "hashslots"
)

const (
	CommandLoginSignIn  = "login.signin"
	CommandLoginSignOut = "login.signout"

	CommandChatUserTalk  = "chat.user.talk"
	CommandChatGroupTalk = "chat.group.talk"
	CommandChatTalkAck   = "chat.talk.ack"

	CommandOfflineIndex   = "chat.offline.index"
	CommandOfflineContent = "chat.offline.content"

	CommandGroupCreate  = "chat.group.create"
	CommandGroupJoin    = "chat.group.join"
	CommandGroupQuit    = "chat.group.quit"
	CommandGroupMembers = "chat.group.members"
	CommandGroupDetail  = "chat.group.detail"
)

// Meta key of packet
const (
	MetaDestServer   = "dest.server"
	MetaDestChannels = "dest.channels"
)

const (
	ProtocolTCP       Protocol = "tcp"
	ProtocolWebsocket Protocol = "websocket"
)

// Service Name
const (
	SNWGateway = "wgateway"
	SNTGateway = "tgateway"
	SNLogin    = "chat"  //login
	SNChat     = "chat"  //chat
	SNService  = "royal" //rpc service
)

type ServiceID string

type SessionID string

const (
	OfflineReadIndexExpiresIn = time.Hour * 24 * 30 // 读索引在缓存中的过期时间
	OfflineSyncIndexCount     = 2000                //单次同步消息索引的数量
	OfflineMessageExpiresIn   = 15                  // 离线消息过期时间
	MessageMaxCountPerPage    = 200                 // 同步消息内容时每页的最大数据
)

const (
	MessageTypeText  = 1
	MessageTypeImage = 2
	MessageTypeVoice = 3
	MessageTypeVideo = 4
)
