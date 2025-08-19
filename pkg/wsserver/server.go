package wsserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/sirupsen/logrus"
	_ "github.com/lypolix/todo-app/docs"
)

type WSServer interface {
	Start() error
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

type Todo struct {
	ID       string    `json:"id"`
	Task     string    `json:"task"`
	Deadline time.Time `json:"deadline"`
	Done     bool      `json:"done"`
}

type Notification struct {
	Type     string    `json:"type"`
	Task     string    `json:"task"`
	Deadline time.Time `json:"deadline"`
	Message  string    `json:"message"`
}

type wsSrv struct {
	router  *gin.Engine
	wsUpg   *websocket.Upgrader
	clients map[*Client]bool
	todos   map[string]Todo
	mu      sync.Mutex
}

func NewWsServer(addr string) WSServer {
	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1"})

	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.File("./web/templates/html/index.html")
	})

	return &wsSrv{
		router:  r,
		wsUpg:   upgrader,
		clients: make(map[*Client]bool),
		todos:   make(map[string]Todo),
	}
}

func (ws *wsSrv) Start() error {
	ws.router.GET("/ws", ws.wsHandler)
	ws.router.GET("/test", ws.testHandler)
	ws.router.POST("/api/todos", ws.createTodoHandler)

	go ws.checkDeadlines()

	logrus.Infof("Starting server on :8000")
	return ws.router.Run(":8000")
}

// wsHandler godoc
// @Summary WebSocket endpoint
// @Description Establish WebSocket connection
// @Tags websocket
// @Schemes ws
// @Success 101 {string} string "Switching Protocols"
// @Router /ws [get]
func (ws *wsSrv) wsHandler(c *gin.Context) {
	conn, err := ws.wsUpg.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}

	ws.mu.Lock()
	ws.clients[client] = true
	ws.mu.Unlock()

	go ws.writePump(client)
	go ws.readPump(client)

	ws.sendTodos(client)
}

func (ws *wsSrv) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
		ws.mu.Lock()
		delete(ws.clients, client)
		ws.mu.Unlock()
	}()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logrus.Errorf("Write error: %v", err)
				return
			}
		case <-ticker.C:
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (ws *wsSrv) readPump(client *Client) {
	defer func() {
		client.conn.Close()
		ws.mu.Lock()
		delete(ws.clients, client)
		ws.mu.Unlock()
	}()

	client.conn.SetReadLimit(512)
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				logrus.Errorf("Read error: %v", err)
			}
			break
		}

		ws.handleMessage(client, message)
	}
}

func (ws *wsSrv) handleMessage(client *Client, message []byte) {
	var msg struct {
		Action string `json:"action"`
		TodoID string `json:"todoId"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		logrus.Errorf("Error parsing message: %v", err)
		return
	}

	switch msg.Action {
	case "complete":
		ws.completeTodo(msg.TodoID)
	}
}

func (ws *wsSrv) sendTodos(client *Client) {
	ws.mu.Lock()
	todos := make([]Todo, 0, len(ws.todos))
	for _, todo := range ws.todos {
		todos = append(todos, todo)
	}
	ws.mu.Unlock()

	message, err := json.Marshal(map[string]interface{}{
		"type":  "todos",
		"todos": todos,
	})
	if err != nil {
		logrus.Errorf("Error marshaling todos: %v", err)
		return
	}

	client.send <- message
}

func (ws *wsSrv) broadcastTodos() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	todos := make([]Todo, 0, len(ws.todos))
	for _, todo := range ws.todos {
		todos = append(todos, todo)
	}

	message, err := json.Marshal(map[string]interface{}{
		"type":  "todos",
		"todos": todos,
	})
	if err != nil {
		logrus.Errorf("Error marshaling todos: %v", err)
		return
	}

	for client := range ws.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(ws.clients, client)
		}
	}
}

func (ws *wsSrv) sendNotification(notificationType, task string, deadline time.Time, message string) {
	notification := Notification{
		Type:     notificationType,
		Task:     task,
		Deadline: deadline,
		Message:  message,
	}

	msg, err := json.Marshal(notification)
	if err != nil {
		logrus.Errorf("Error marshaling notification: %v", err)
		return
	}

	ws.mu.Lock()
	defer ws.mu.Unlock()

	for client := range ws.clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(ws.clients, client)
		}
	}
}

func (ws *wsSrv) checkDeadlines() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ws.mu.Lock()
			for id, todo := range ws.todos {
				if todo.Done {
					continue
				}

				remaining := time.Until(todo.Deadline)
				if remaining <= 0 {
					ws.sendNotification("deadline_passed", todo.Task, todo.Deadline, "Deadline has passed!")
					todo.Done = true
					ws.todos[id] = todo
				} else if remaining <= 30*time.Minute {
					ws.sendNotification("deadline_soon", todo.Task, todo.Deadline,
						"Deadline is approaching! "+formatDuration(remaining))
				}
			}
			ws.mu.Unlock()
		}
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return h.String() + "h" + m.String() + "m"
}

func (ws *wsSrv) createTodoHandler(c *gin.Context) {
	var todo struct {
		Task     string    `json:"task"`
		Deadline time.Time `json:"deadline"`
	}

	if err := c.ShouldBindJSON(&todo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newTodo := Todo{
		Task:     todo.Task,
		Deadline: todo.Deadline,
		Done:     false,
	}

	ws.addTodo(newTodo)
	c.JSON(http.StatusCreated, newTodo)
}

func (ws *wsSrv) addTodo(todo Todo) {
	ws.mu.Lock()
	todo.ID = generateID()
	ws.todos[todo.ID] = todo
	ws.mu.Unlock()

	ws.broadcastTodos()
	go ws.checkDeadline(todo.ID)
}

func (ws *wsSrv) completeTodo(todoID string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if todo, exists := ws.todos[todoID]; exists {
		todo.Done = true
		ws.todos[todoID] = todo
		ws.broadcastTodos()
	}
}

func (ws *wsSrv) checkDeadline(todoID string) {
	ws.mu.Lock()
	todo, exists := ws.todos[todoID]
	ws.mu.Unlock()

	if !exists || todo.Done {
		return
	}

	for {
		time.Sleep(1 * time.Minute)

		ws.mu.Lock()
		todo, exists = ws.todos[todoID]
		if !exists || todo.Done {
			ws.mu.Unlock()
			return
		}

		remaining := time.Until(todo.Deadline)
		if remaining <= 0 {
			ws.sendNotification("deadline_passed", todo.Task, todo.Deadline, "Deadline has passed!")
			todo.Done = true
			ws.todos[todoID] = todo
			ws.mu.Unlock()
			return
		} else if remaining <= 30*time.Minute {
			ws.sendNotification("deadline_soon", todo.Task, todo.Deadline,
				"Deadline is approaching! "+formatDuration(remaining))
		}
		ws.mu.Unlock()
	}
}

func generateID() string {
	return time.Now().Format("20060102150405")
}

// testHandler godoc
// @Summary Test endpoint
// @Description Simple test endpoint
// @Tags test
// @Accept json
// @Produce json
// @Success 200 {string} string "Test is successful"
// @Router /test [get]
func (ws *wsSrv) testHandler(c *gin.Context) {
	c.String(http.StatusOK, "Test is successful")
}