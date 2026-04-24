package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"time"

	"charm.land/fantasy"
)

type HandshakeState int

const (
	HandshakeStateStatus HandshakeState = iota
	HandshakeStateLogin
	HandshakeStateTransfer
)

type ClientHandshake struct {
	ProtocolVersion int32 `encode:"varint"`
	ServerAddress   string
	ServerPort      uint16
	NextState       HandshakeState `encode:"varint"`
}

type ClientStatusRequest struct{}

type ForgeMessageChannel struct {
	Resource string `json:"res"`
	Version  string `json:"version"`
	Required bool   `json:"required"`
}

type ForgeMod struct {
	ModId     string `json:"modId"`
	ModMarker string `json:"modmarker"`
}

type ForgeData struct {
	Channels       []ForgeMessageChannel `json:"channels"`
	Mods           []ForgeMod            `json:"mods"`
	NetworkVersion int                   `json:"fmlNetworkVersion"`
}

type ForgeModInfo struct {
	Type    string `json:"type"`
	ModList []struct {
		ModId   string `json:"modId"`
		Version string `json:"version"`
	}
}

type StatusResponse struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int32  `json:"protocol"`
	} `json:"version"`

	Players struct {
		Max    int32 `json:"max"`
		Online int32 `json:"online"`
		Sample []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"sample,omitempty"`
	} `json:"players"`

	Description        interface{}   `json:"description"` // Can be string or struct
	Favicon            string        `json:"favicon"`
	EnforcesSecureChat bool          `json:"enforcesSecureChat"`
	ForgeData          *ForgeData    `json:"forgeData"`
	ModInfo            *ForgeModInfo `json:"modinfo"`
}

type ServerStatusResponse struct {
	Response *StatusResponse `encode:"json"`
}

const (
	SegmentBits uint32 = 0x7F
	ContinueBit uint32 = 0x80
)

func writeVarInt32(w io.Writer, num int32) error {
	val := uint32(num)
	var buf []byte

	for {
		// Take the lower 7 bits

		if (val & ^SegmentBits) == 0 {
			_, err := w.Write([]byte{byte(val)})
			return err
		}

		w.Write([]byte{byte((val & SegmentBits) | ContinueBit)})

		val >>= 7

		if val == 0 {
			break
		}
	}

	_, err := w.Write(buf)
	return err
}

func readUnsignedVarInt32From(r io.ByteReader) (int, uint32) {
	val, _ := binary.ReadUvarint(r)

	buf := bytes.NewBuffer([]byte{})
	writeVarInt32(buf, int32(val))
	return buf.Len(), uint32(val)
}

func readUnsignedVarInt32(b []byte) uint32 {
	val, _ := binary.ReadUvarint(bytes.NewReader(b))
	return uint32(val)
}

// pack structure is
// length(packet id) + length(data)
// packet id
// data
func packPacket(packetId uint32, data []byte) []byte {
	packetBuf := bytes.NewBuffer([]byte{})

	packetIdBuf := bytes.NewBuffer([]byte{})
	writeVarInt32(packetIdBuf, int32(packetId))

	writeVarInt32(packetBuf, int32(packetIdBuf.Len()+len(data)))

	// re-use what was already written
	packetBuf.Write(packetIdBuf.Bytes())

	packetBuf.Write(data)

	return packetBuf.Bytes()
}

func packString(str string) []byte {
	// a string in minecraft is encoded by var int length + string
	encodedStringBuffer := bytes.NewBuffer([]byte{})

	writeVarInt32(encodedStringBuffer, int32(len([]byte(str))))
	// binary.Write(encodedStringBuffer, binary.BigEndian, []byte(str))
	encodedStringBuffer.Write([]byte(str))

	return encodedStringBuffer.Bytes()
}

func packHandshake(host, port string) []byte {
	buf := bytes.NewBuffer([]byte{})

	if port == "" {
		port = "25565"
	}

	nPort, _ := strconv.Atoi(port)

	writeVarInt32(buf, -1)
	buf.Write(packString(host))
	binary.Write(buf, binary.BigEndian, uint16(nPort))
	writeVarInt32(buf, 1)

	return packPacket(0x00, buf.Bytes())
}

func readString(b []byte) (string, error) {
	reader := bytes.NewReader(b)
	strLength, err := binary.ReadUvarint(reader)

	if err != nil {
		return "", err
	}

	buf := make([]byte, strLength)

	read, err := reader.Read(buf)

	if err != nil {
		return "", err
	}

	if read != int(strLength) {
		return "", errors.New("length mismatch")
	}

	return string(buf), nil
}

type packet struct {
	packetId   int
	packetData []byte
}

func readPacketData(b []byte) packet {
	buf := bytes.NewBuffer(b)
	// read packet length
	readUnsignedVarInt32From(buf)

	_, packetId := readUnsignedVarInt32From(buf)
	return packet{
		packetId:   int(packetId),
		packetData: buf.Bytes(),
	}
}

func readPacketDataFrom(r io.Reader) (packet, error) {
	packetReader := bufio.NewReader(r)
	// read packet length
	_, length := readUnsignedVarInt32From(packetReader)
	size, packetId := readUnsignedVarInt32From(packetReader)

	if length == 0 {
		return packet{
			packetId:   int(packetId),
			packetData: []byte{},
		}, nil
	}
	packetData := make([]byte, int(length)-size)

	_, err := io.ReadFull(packetReader, packetData)
	return packet{
		packetId:   int(packetId),
		packetData: packetData,
	}, err
}

func lookupHost(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)

	if err != nil {
		// try looking up host
		_, srv, err := net.LookupSRV("minecraft", "tcp", addr)

		if err != nil {
			// no service found, let's try setting the host port to address & 25565
			host = addr
			port = "25565"
		}

		if len(srv) > 0 {
			// use the highest sorted service
			return net.JoinHostPort(srv[0].Target, strconv.Itoa(int(srv[0].Port))), nil
		}

	}

	return net.JoinHostPort(host, port), nil
}

func Ping(addr string) (*StatusResponse, error) {

	addr, err := lookupHost(addr)

	if err != nil {
		return nil, err
	}

	host, port, _ := net.SplitHostPort(addr)

	conn, err := net.DialTimeout("tcp", addr, 1000*time.Millisecond)

	if err != nil {
		return nil, err
	}

	conn.SetDeadline(time.Now().Add(1000 * time.Millisecond))
	if _, err = conn.Write(packHandshake(host+"\x00FML2\x00", port)); err != nil {
		return nil, err
	}

	if _, err = conn.Write(packPacket(0x00, []byte{})); err != nil {
		return nil, err
	}

	unparsedResponsePacket, err := readPacketDataFrom(conn)

	if err != nil {
		conn.Close()
		return nil, err
	}

	unparsedResponse, err := readString(unparsedResponsePacket.packetData)

	if err != nil {
		return nil, err
	}

	var statusResponse StatusResponse

	if err = json.Unmarshal([]byte(unparsedResponse), &statusResponse); err != nil {
		return nil, err
	}

	return &statusResponse, nil
}

func MinecraftPingerTool() fantasy.AgentTool {
	return fantasy.NewAgentTool[string](
		"minecraft_pinger",
		"ping minecraft servers for lively data",
		func(ctx context.Context, input string, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			statusResponse, err := Ping(input)

			if err != nil {
				return fantasy.ToolResponse{}, err
			}

			return fantasy.ToolResponse{
				Content: `
Server Version: ` + statusResponse.Version.Name + `
Protocol Version: ` + strconv.Itoa(int(statusResponse.Version.Protocol)) + `
Players Online: ` + strconv.Itoa(int(statusResponse.Players.Online)) + ` / ` + strconv.Itoa(int(statusResponse.Players.Max)) + `
Description: ` + func() string {
					switch desc := statusResponse.Description.(type) {
					case string:
						return desc
					case map[string]interface{}:
						if text, ok := desc["text"].(string); ok {
							return text
						}
						return ""
					default:
						return ""
					}
				}(),
			}, nil
		},
	)
}
