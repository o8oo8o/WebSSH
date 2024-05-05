package service

import (
	"gossh/app/config"
	"gossh/gin"
	"gossh/gin/sse"
	"net/http"
	"sort"
	"time"
)

type SshConnById []SshConn

func (a SshConnById) Len() int           { return len(a) }
func (a SshConnById) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SshConnById) Less(i, j int) bool { return a[i].SessionId < a[j].SessionId }

func GetOnlineClient(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Connection", "keep-alive")
	c.Header("Cache-Control", "no-cache")
	c.Header("Content-Type", "text/event-stream")

	for {
		c.Writer.(http.Flusher).Flush()
		time.Sleep(config.DefaultConfig.StatusRefresh)
		select {
		case <-c.Request.Context().Done():
			return
		default:
			var data SshConnById
			OnlineClients.Range(func(key, value any) bool {
				conn, ok := value.(*SshConn)
				if ok && conn != nil {
					data = append(data, *conn)
				}
				return ok
			})

			sort.Sort(data)
			c.Render(200, sse.Event{
				Id:    "200",
				Event: "message",
				Retry: 10000,
				Data: map[string]any{
					"code": 0,
					"data": data,
					"msg":  "ok",
				},
			})
		}
	}
}

func RefreshConnTime(c *gin.Context) {
	ids := c.PostFormArray("ids")
	for _, key := range ids {
		cli, ok := OnlineClients.Load(key)
		if !ok {
			continue
		}
		conn, ok := cli.(*SshConn)
		if conn != nil && ok {
			conn.LastActiveTime = time.Now()
		}
	}
	c.JSON(200, gin.H{
		"code": 0,
		"data": ids,
		"msg":  "ok",
	})
}
