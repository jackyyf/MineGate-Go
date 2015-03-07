package realip

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"github.com/jackyyf/MineGate-Go/minegate"
)

func init() {
	minegate.OnLoginRequest(HandleLogin, 0)
}

const prefix = "OfflinePlayer:"

func ToUUID(h [16]byte) (uuid string) {
	h[6] &= 0x0F
	h[6] |= 0x30
	h[8] &= 0x3F
	h[8] |= 0x80
	return hex.EncodeToString(h[:4]) + "-" + hex.EncodeToString(h[4:6]) + "-" + hex.EncodeToString(h[6:8]) +
		"-" + hex.EncodeToString(h[8:10]) + "-" + hex.EncodeToString(h[10:])
}

func HandleLogin(lre *minegate.LoginRequestEvent) {
	if lre.Rejected() {
		return
	}
	res, err := lre.Upstream.GetExtra("bungeecord")
	if err != nil {
		//Assume false.
		return
	}
	bval := minegate.ToBool(res)
	if bval {
		// Enabled bungeecord support.
		lre.Infof("Patching for bungeecord.")
		uname := lre.LoginPacket.Name
		remoteip := lre.GetRemoteIP()
		buff := bytes.NewBuffer(make([]byte, 0, len(prefix)+len(uname)+4))
		buff.WriteString(prefix)
		buff.WriteString(uname)
		// Data is the faked uuid, for offline only.
		// Online mode is not available, since online mode introduces protocol encryption.
		// For online mode, please use bungeecord!
		data := ToUUID(md5.Sum(buff.Bytes()))
		lre.Debugf("Offline username: %s, UUID: %s", uname, data)
		lre.InitPacket.ServerAddr += "\x00" + remoteip + "\x00" + data
	}
	return
}
