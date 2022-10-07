package pkt

import (
	"github.com/golang/protobuf/proto"
	"github.com/wangxuefeng90923/wxf/wire"
	"github.com/wangxuefeng90923/wxf/wire/endian"
	"io"
	"strconv"
	"strings"
)

type LogicPkt struct {
	Header
	Body []byte `json:"body,omitempty"`
}

type HeaderOption func(*Header)

func (p *LogicPkt) Decode(r io.Reader) error {
	headerBytes, err := endian.ReadBytes(r)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(headerBytes, &p.Header)
	if err != nil {
		return err
	}
	p.Body, err = endian.ReadBytes(r)
	if err != nil {
		return err
	}
	return nil
}

func (p *LogicPkt) Encode(w io.Writer) error {
	headerBytes, err := proto.Marshal(&p.Header)
	if err != nil {
		return err
	}
	if err := endian.WriteBytes(w, headerBytes); err != nil {
		return err
	}
	if err := endian.WriteBytes(w, p.Body); err != nil {
		return err
	}
	return nil
}

func (p *LogicPkt) GetMeta(key string) (any, bool) {
	return FindMeta(p.Meta, key)
}

func FindMeta(meta []*Meta, key string) (any, bool) {
	for _, m := range meta {
		if m.Key == key {
			switch m.Type {
			case MetaType_int:
				v, _ := strconv.Atoi(m.Value)
				return v, true
			case MetaType_float:
				v, _ := strconv.ParseFloat(m.Value, 64)
				return v, true
			}
			return m.Value, true
		}
	}
	return nil, false
}

func (p *LogicPkt) DelMeta(key string) {
	for i, m := range p.Meta {
		if m.Key == key {
			length := len(p.Meta)
			if i < length-1 {
				copy(p.Meta[i:], p.Meta[i+1:])
			}
			p.Meta = p.Meta[:length-1]
		}
	}
}

func (p *LogicPkt) AddStringMeta(key, value string) {
	p.AddMeta(&Meta{
		Key:   key,
		Value: value,
		Type:  MetaType_string,
	})
}

func (p *LogicPkt) AddMeta(m ...*Meta) {
	p.Meta = append(p.Meta, m...)
}

func (x *Header) ServiceName() string {
	arr := strings.SplitN(x.Command, ".", 2)
	if len(arr) <= 1 {
		return "default"
	}
	return arr[0]
}

func New(command string, options ...HeaderOption) *LogicPkt {
	pkt := &LogicPkt{}
	pkt.Command = command

	for _, option := range options {
		option(&pkt.Header)
	}
	if pkt.Sequence == 0 {
		pkt.Sequence = wire.Seq.Next()
	}
	return pkt
}

func NewFrom(header *Header) *LogicPkt {
	pkt := &LogicPkt{}
	pkt.Header = Header{
		Command:   header.Command,
		Sequence:  header.Sequence,
		ChannelId: header.ChannelId,
		Status:    header.Status,
		Dest:      header.Dest,
	}
	return pkt
}

func (p *LogicPkt) ReadBody(val proto.Message) error {
	return proto.Unmarshal(p.Body, val)
}

func (p *LogicPkt) WriteBody(val proto.Message) *LogicPkt {
	if val == nil {
		return p
	}
	p.Body, _ = proto.Marshal(val)
	return p
}

func WithStatus(status Status) HeaderOption {
	return func(h *Header) {
		h.Status = status
	}
}

func WithSeq(seq uint32) HeaderOption {
	return func(h *Header) {
		h.Sequence = seq
	}
}

func WithChannel(channelID string) HeaderOption {
	return func(h *Header) {
		h.ChannelId = channelID
	}
}

func WithDest(dest string) HeaderOption {
	return func(h *Header) {
		h.Dest = dest
	}
}
