package realip

import (
	"bytes"
	"strings"
	"crypto/md5"
	"encoding/hex"
	"github.com/jackyyf/MineGate-Go/minegate"
	log "github.com/jackyyf/golog"
)

func init() {
	minegate.OnLoginRequest(HandleLogin, 0)
}

func ToUUID(h [16]byte) (uuid string) {
	h[6] &= 0x0F
	h[6] |= 0x30
	h[8] &= 0x3F
	h[8] |= 0x80
	return hex.EncodeToString(h[:4]) + "-" + hex.EncodeToString(h[4:6]) + "-" + hex.EncodeToString(h[6:8]) +
		"-" + hex.EncodeToString(h[8:10]) + "-" + hex.EncodeToString(h[10:])
}

func HandleLogin(lre *minegate.LoginRequestEvent) {
	res, err := lre.Upstream.GetExtra("bungeecord")
	if err != nil {
		//Assume false.
		return
	}
	bval, ok := res.(bool)
	if !ok {
		ival, ok := res.(int)
		if !ok {
			sval, ok := res.(string)
			if !ok {
				return
			} else {
				sval = strings.ToLower(sval)
				bval = sval == "true" || sval == "yes" || sval == "on" || sval == "y";
			}
		} else {
			bval = ival > 0
		}
	}
	if bval {
		// Enabled bungeecord support.
		log.Infof("Patching for bungeecord.")
		uname := lre.LoginPacket.Name
		remoteip := lre.GetRemoteIP()
		prefix := "OfflinePlayer:"
		buff := bytes.NewBuffer(make([]byte, 0, len(prefix)+len(uname)+4))
		buff.WriteString(prefix)
		buff.WriteString(uname)
		// Data is the faked uuid, for offline only.
		// Online mode is not available, since online mode introduces protocol encryption.
		// For online mode, please use bungeecord!
		data := ToUUID(md5.Sum(buff.Bytes()))
		log.Debugf("UUID: %s", data)
		lre.InitPacket.ServerAddr += "\x00" + remoteip + "\x00" + data
	}
	return
}
