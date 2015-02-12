package mcchat

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
)

/**
 * Note, this chat msg struct is not full compatiable to minecraft.
 * Only part of its features are implemented, thus can be used to
 * response to client as ping result.
 */

type Color string

var color_string = [...]string{
	"black",
	"dark_blue",
	"dark_green",
	"dark_aqua",
	"dark_red",
	"dark_purple",
	"gold",
	"gray",
	"dark_gray",
	"blue",
	"green",
	"aqua",
	"red",
	"light_purple",
	"yellow",
	"white",
}

type Event struct {
	Action string
	Value  string
}

type Style int32

const (
	BOLD Style = 1 << iota
	ITALIC
	UNDERLINED
	STRIKETHROUGH
	BLACK        Style = 16
	DARK_BLUE    Style = 48
	DARK_GREEN   Style = 80
	DARK_AQUA    Style = 112
	DARK_RED     Style = 144
	DARK_PURPLE  Style = 176
	GOLD         Style = 208
	GRAY         Style = 240
	DARK_GRAY    Style = 272
	BLUE         Style = 304
	GREEN        Style = 336
	AQUA         Style = 368
	RED          Style = 400
	LIGHT_PURPLE Style = 432
	YELLOW       Style = 464
	WHITE        Style = 496
	RESET        Style = -1
)

type ChatMsg struct {
	Text          string
	Bold          bool       `json:",omitempty"`
	Italic        bool       `json:",omitempty"`
	Underlined    bool       `json:",omitempty"`
	Strikethrough bool       `json:",omitempty"`
	Color         string     `json:",omitempty"`
	OnClick       *Event     `json:"clickEvent,omitempty"`
	OnHover       *Event     `json:"hoverEvent,omitempty"`
	ExtraMsg      []*ChatMsg `json:"extra,omitempty"`
}

func NewMsg(msg string) (chatmsg *ChatMsg) {
	return &ChatMsg{
		Text: msg,
	}
}

func (msg *ChatMsg) SetStyle(style_mask Style) {
	if style_mask == -1 {
		// Reset
		msg.Bold = false
		msg.Italic = false
		msg.Underlined = false
		msg.Strikethrough = false
		msg.Color = "reset"
		return
	}
	if (style_mask & 16) != 0 {
		// Set color
		color_index := (style_mask >> 5) & 15
		msg.Color = color_string[color_index]
	}
	msg.Bold = (style_mask & BOLD) == 1
	msg.Italic = (style_mask & ITALIC) == 1
	msg.Underlined = (style_mask & UNDERLINED) == 1
	msg.Strikethrough = (style_mask & STRIKETHROUGH) == 1
}

func (msg *ChatMsg) SetBold(bold bool) {
	msg.Bold = bold
}

func (msg *ChatMsg) SetItalic(italic bool) {
	msg.Italic = italic
}

func (msg *ChatMsg) SetUnderlined(underlined bool) {
	msg.Underlined = underlined
}

func (msg *ChatMsg) SetStrikeThrough(strikethrough bool) {
	msg.Strikethrough = strikethrough
}

func (msg *ChatMsg) SetColor(color Style) {
	if color&16 == 0 {
		return
	}
	color = (color >> 5) & 15
	msg.Color = color_string[color]
}

func (msg *ChatMsg) HoverMsg(message string) {
	if message == "" {
		msg.OnHover = nil
	} else {
		msg.OnHover = &Event{
			Action: "show_text",
			Value:  message,
		}
	}
}

func (msg *ChatMsg) ClickTarget(target string) {
	if target == "" {
		msg.OnClick = nil
	} else {
		msg.OnClick = &Event{
			Action: "open_url",
			Value:  target,
		}
	}
}

func (msg *ChatMsg) AsJson() (json_data []byte) {
	json_data, _ = json.Marshal(msg)
	return
}

func (msg *ChatMsg) AsChatString() (chat_string []byte) {
	json_data := msg.AsJson()
	json_length := len(json_data)
	buffer := bytes.NewBuffer(make([]byte, 0, json_length+binary.MaxVarintLen32))
	minor_buffer := make([]byte, binary.MaxVarintLen32)
	buffer.Write(minor_buffer[:binary.PutUvarint(minor_buffer, uint64(json_length))])
	buffer.Write(json_data)
	return buffer.Bytes()
}
