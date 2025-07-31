package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/thanhpk/randstr"

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

func StatusWS(c *gin.Context) {
	ws, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		if err.Error() == "websocket: client sent data before handshake is complete" {
			c.Abort()
			_ = c.Error(err)
			return
		}
		return
	}

	socketCtx, cancel := context.WithCancelCause(c)

	ws.SetPingHandler(nil) // use the default values provided by the package
	ws.SetPongHandler(nil) // use the default values provided by the package
	ws.SetCloseHandler(func(code int, text string) error {
		message := websocket.FormatCloseMessage(code, text)
		_ = ws.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		cancel(errors.New("recevied close message"))
		_ = ws.Close()
		return nil
	})

	binaryMessages := make(chan v1.BinaryMessage)
	textMessages := make(chan v1.TextMessage)

	_, _ = startReceivingMessages(socketCtx, ws, binaryMessages, textMessages)

	t := time.NewTicker(5 * time.Second)

	pingMessage, _ := websocket.NewPreparedMessage(websocket.PingMessage, []byte("hello there"))

	go func() {
		for {
			select {
			case <-socketCtx.Done():
				return
			case msg := <-binaryMessages:
				fmt.Println("[Received Binary Message]", msg.ReceivedAt, msg.Content)
			case msg := <-textMessages:
				fmt.Println("[Received Text Message]", msg.ReceivedAt, msg.Content)
			case <-t.C:
				var statuses []v1.ServiceStatus

				for range 10 {
					statuses = append(statuses, v1.ServiceStatus{
						Path:       randstr.Hex(12),
						Status:     "TESTING",
						LastUpdate: time.Now(),
					})
				}
				_ = ws.WritePreparedMessage(pingMessage)

				_ = ws.WriteJSON(statuses)
			}
		}
	}()
}

func startReceivingMessages(ctx context.Context, ws *websocket.Conn, b chan v1.BinaryMessage, t chan v1.TextMessage) (context.Context, context.CancelCauseFunc) { //nolint:lll
	receiverContext, cancel := context.WithCancelCause(ctx)

	go func() {
		for {
			select {
			case <-receiverContext.Done():
				return
			default:
				messageType, message, err := ws.ReadMessage()
				if err != nil {
					cancel(err)
					return
				}

				switch messageType {
				case websocket.BinaryMessage:
					b <- v1.BinaryMessage{Content: message, ReceivedAt: time.Now()}
				case websocket.TextMessage:
					t <- v1.TextMessage{Content: string(message), ReceivedAt: time.Now()}
				}
			}
		}
	}()

	return receiverContext, cancel
}
