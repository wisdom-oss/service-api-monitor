package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	wisdomTypes "github.com/wisdom-oss/common-go/v3/types"

	"microservice/traefik"
	v1 "microservice/types/v1"
	"microservice/types/v1/commands"
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

	var t *time.Ticker
	t = &time.Ticker{}

	pingMessage, _ := websocket.NewPreparedMessage(websocket.PingMessage, []byte("hello there"))

	go func() {
		var command v1.Command
		for {
			select {
			case <-socketCtx.Done():
				return
			case msg := <-binaryMessages:
				err := json.Unmarshal(msg.Content, &command)
				if err != nil {
					ws.CloseHandler()(websocket.CloseInternalServerErr, err.Error())
					return
				}
			case msg := <-textMessages:
				err := json.Unmarshal([]byte(msg.Content), &command)
				if err != nil {
					ws.CloseHandler()(websocket.CloseInternalServerErr, err.Error())
					return
				}
			case <-t.C:
				var statuses []v1.ServiceStatus

				statuses, err := traefik.ServiceStatus("/api/files")
				if err != nil {
					ws.CloseHandler()(websocket.CloseInternalServerErr, err.Error())
					return
				}
				_ = ws.WritePreparedMessage(pingMessage)
				_ = ws.WriteJSON(statuses)
				continue
			}

			if err := command.Validate(); err != nil {
				ws.CloseHandler()(websocket.ClosePolicyViolation, err.Error())
				return
			}

			switch command.Command {
			case "subscribe":
				var data commands.Subscribe
				err := json.Unmarshal(command.Data, &data)
				if err != nil {
					ws.CloseHandler()(websocket.ClosePolicyViolation, err.Error())
					return
				}

				if err := data.Validate(); err != nil {
					ws.CloseHandler()(websocket.ClosePolicyViolation, err.Error())
					return
				}

				if data.Interval.ToTimeDuration() == time.Duration(0) {
					t = time.NewTicker(15 * time.Second)
				} else {
					t = time.NewTicker(data.Interval.ToTimeDuration())
				}

				statuses, err := traefik.ServiceStatus(data.Paths...)
				if err != nil {
					ws.CloseHandler()(websocket.CloseInternalServerErr, err.Error())
					return
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
