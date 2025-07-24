package v1

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	wisdomTypes "github.com/wisdom-oss/common-go/v3/types"

	v1 "microservice/types/v1"
)

const bufferSizeLimit = 2048

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  bufferSizeLimit,
	WriteBufferSize: bufferSizeLimit,
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		err := wisdomTypes.ServiceError{
			Title:  "Websocket Failure",
			Status: uint(status),
			Detail: reason.Error(),
			Type:   "https://www.rfc-editor.org/rfc/rfc6455.html#section-7.4.1",
		}
		err.Emit(w)
	},
}

type Message struct {
	MessageType int
	Content     []byte
}

func StatusWS(c *gin.Context) {

	updateIntervalMillis, err := strconv.ParseInt(c.DefaultQuery("updateInterval", "15000"), 10, 0)
	if err != nil {
		c.Abort()
		_ = c.Error(err)
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		if err.Error() == "websocket: client sent data before handshake is complete" {
			c.Abort()
			_ = c.Error(err)
			return
		}
		return
	}

	fmt.Println("opened connection to:", conn.RemoteAddr().String())
	_ = conn.WriteJSON(map[string]any{"updatesFor": c.Param("service")})

	incomingPings := make(chan v1.BinaryMessage, 500)
	incomingMessages := make(chan any, 500)
	closing := make(chan bool, 2)
	isClosed := false

	go func() {
		for {
			if isClosed {
				continue
			}
			msgType, content, err := conn.ReadMessage()
			if err != nil {
				isClosed = true
				closing <- true
				if websocket.IsCloseError(err) {
					conn.Close()
					return
				}
			}

			switch msgType {
			case websocket.PongMessage:
				// ignore these frames
				break
			case websocket.CloseMessage:
				fmt.Println("received close message")
				isClosed = true
				closing <- true
				msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "good bye!")
				conn.WriteMessage(websocket.CloseMessage, msg)
				conn.Close()
			case websocket.PingMessage:
				incomingPings <- v1.BinaryMessage{
					Type:    msgType,
					Payload: content,
				}
			case websocket.TextMessage:
				incomingMessages <- v1.TextMessage{
					Type:    msgType,
					Payload: string(content),
				}
			case websocket.BinaryMessage:
				incomingMessages <- v1.BinaryMessage{
					Type:    msgType,
					Payload: content,
				}
			default:
				closing <- true
				msg := websocket.FormatCloseMessage(websocket.CloseProtocolError, "invalid message type received")
				conn.WriteMessage(websocket.CloseMessage, msg)
				isClosed = true
			}
		}
	}()

	var ticker *time.Ticker
	if updateIntervalMillis > 0 {
		ticker = time.NewTicker(time.Duration(updateIntervalMillis * int64(time.Millisecond)))
	} else {
		ticker = time.NewTicker(5 * time.Second)
	}

	for {
		if isClosed {
			break
		}
		select {
		case msg := <-incomingPings:
			_ = conn.WriteMessage(websocket.PongMessage, msg.Payload)
		case incoming := <-incomingMessages:
			switch msg := incoming.(type) {
			case v1.TextMessage:
				fmt.Println("[INCOMING MESSAGE (Text)]", msg.Payload)
			case v1.BinaryMessage:
				fmt.Println("[INCOMING MESSAGE (Binary)]", msg.Payload)
			}
		case t := <-ticker.C:
			conn.WriteMessage(websocket.PingMessage, nil)
			fmt.Println("[PUSHING SERVICE STATUS]", t)
			conn.WriteJSON("update")
		default:
			continue
		}
	}
	conn.Close()

}
